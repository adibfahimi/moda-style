package handlers_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/cart-service/database"
	"github.com/adibfahimi/moda-style/services/cart-service/handlers"
	"github.com/adibfahimi/moda-style/services/cart-service/models"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testJWTSecret is the signing key every test in this package uses. TestMain
// installs it before any test runs because common resolves JWT_SECRET lazily.
const testJWTSecret = "cart-service-test-secret"

// dbCounter hands every test its own in-memory database.
var dbCounter atomic.Uint64

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

// testProduct mirrors the products table that the product service owns but this
// service reads through the shared database.
type testProduct struct {
	ID       uint `gorm:"primaryKey"`
	Name     string
	ImageURL string
	Price    float64
}

// TableName pins the test model onto the shared production table name.
func (testProduct) TableName() string { return "products" }

// testSize mirrors the sizes table of the product service.
type testSize struct {
	ID        uint `gorm:"primaryKey"`
	ProductID uint
	Size      string
	Color     string
	Stock     int
}

// TableName pins the test model onto the shared production table name.
func (testSize) TableName() string { return "sizes" }

// newTestDB returns an isolated in-memory SQLite database with the cart schema
// plus the product/size tables migrated, and points database.DB at it.
//
// SQLite keeps the handler tests self-contained; the SQL used here (simple
// SELECTs, SUM and soft deletes) behaves identically on PostgreSQL.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:cart_test_%d?mode=memory&cache=shared", dbCounter.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("opening in-memory database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("accessing database handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&models.CartItem{}, &models.WishlistItem{}, &testProduct{}, &testSize{}); err != nil {
		t.Fatalf("migrating cart schema: %v", err)
	}

	database.DB = db
	return db
}

// newTestApp mounts the same routes as services/cart-service/main.go.
func newTestApp() *fiber.App {
	app := fiber.New()

	api := app.Group("/api/v1")
	api.Get("/cart", common.RequireAuth, handlers.GetCart)
	api.Post("/cart", common.RequireAuth, handlers.AddToCart)
	api.Put("/cart/:id", common.RequireAuth, handlers.UpdateCartItem)
	api.Delete("/cart/:id", common.RequireAuth, handlers.RemoveFromCart)
	api.Delete("/cart", common.RequireAuth, handlers.ClearCart)
	api.Get("/wishlist", common.RequireAuth, handlers.GetWishlist)
	api.Post("/wishlist/:product_id", common.RequireAuth, handlers.ToggleWishlistItem)

	return app
}

// request drives app with an HTTP request and returns the status code and raw
// body. A non-empty token is sent as a bearer credential.
func request(t *testing.T, app *fiber.App, method, path, body, token string) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}

	return resp.StatusCode, raw
}

// decodeJSON unmarshals a JSON object body into a generic map.
func decodeJSON(t *testing.T, raw []byte) map[string]interface{} {
	t.Helper()

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("response is not valid JSON (%q): %v", string(raw), err)
	}
	return decoded
}

// stringField returns the named string field, failing the test when it is
// missing or of another type.
func stringField(t *testing.T, body map[string]interface{}, key string) string {
	t.Helper()

	value, ok := body[key].(string)
	if !ok {
		t.Fatalf("field %q is missing or not a string in %v", key, body)
	}
	return value
}

// sliceField returns the named JSON array, failing the test when it is missing.
func sliceField(t *testing.T, body map[string]interface{}, key string) []interface{} {
	t.Helper()

	value, ok := body[key].([]interface{})
	if !ok {
		t.Fatalf("field %q is missing or not an array in %v", key, body)
	}
	return value
}

// numberField returns the named numeric field as a float64.
func numberField(t *testing.T, body map[string]interface{}, key string) float64 {
	t.Helper()

	value, ok := body[key].(float64)
	if !ok {
		t.Fatalf("field %q is missing or not a number in %v", key, body)
	}
	return value
}

// objectField returns the named nested object as a generic map.
func objectField(t *testing.T, body map[string]interface{}, key string) map[string]interface{} {
	t.Helper()

	value, ok := body[key].(map[string]interface{})
	if !ok {
		t.Fatalf("field %q is missing or not an object in %v", key, body)
	}
	return value
}

// tokenFor issues a valid access token for the given identity so tests can call
// the routes that sit behind common.RequireAuth.
func tokenFor(t *testing.T, userID uint) string {
	t.Helper()

	token, err := common.GenerateToken(userID, fmt.Sprintf("user%d@example.com", userID), "Ada", "user")
	if err != nil {
		t.Fatalf("generating token: %v", err)
	}
	return token
}

// seededCart is the catalogue fixture plus the sizes tests shop from.
type seededCart struct {
	dressID     uint
	dressSizeID uint // M/Black, three in stock
	soldOutID   uint // L/Black, no stock
	coatID      uint
	coatSizeID  uint // XL/Grey, two in stock
}

// seedCatalogue inserts two products with their size variants so the cart
// handlers can resolve names, prices and stock.
func seedCatalogue(t *testing.T, db *gorm.DB) seededCart {
	t.Helper()

	dress := testProduct{Name: "Silk Midi Dress", ImageURL: "dress.png", Price: 89.5}
	if err := db.Create(&dress).Error; err != nil {
		t.Fatalf("creating product: %v", err)
	}
	coat := testProduct{Name: "Wool Coat", ImageURL: "coat.png", Price: 180}
	if err := db.Create(&coat).Error; err != nil {
		t.Fatalf("creating product: %v", err)
	}

	createSize := func(productID uint, size, color string, stock int) testSize {
		variant := testSize{ProductID: productID, Size: size, Color: color, Stock: stock}
		if err := db.Create(&variant).Error; err != nil {
			t.Fatalf("creating size: %v", err)
		}
		return variant
	}

	return seededCart{
		dressID:     dress.ID,
		dressSizeID: createSize(dress.ID, "M", "Black", 3).ID,
		soldOutID:   createSize(dress.ID, "L", "Black", 0).ID,
		coatID:      coat.ID,
		coatSizeID:  createSize(coat.ID, "XL", "Grey", 2).ID,
	}
}

// addBody renders the JSON payload accepted by AddToCart.
func addBody(productID, sizeID uint, quantity int) string {
	return fmt.Sprintf(`{"product_id":%d,"size_id":%d,"quantity":%d}`, productID, sizeID, quantity)
}

func TestCartRoutesRequireAuthentication(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/cart"},
		{http.MethodPost, "/api/v1/cart"},
		{http.MethodPut, "/api/v1/cart/1"},
		{http.MethodDelete, "/api/v1/cart/1"},
		{http.MethodDelete, "/api/v1/cart"},
		{http.MethodGet, "/api/v1/wishlist"},
		{http.MethodPost, "/api/v1/wishlist/1"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			status, raw := request(t, app, route.method, route.path, "", "")
			if status != fiber.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized, got %d: %s", status, raw)
			}
			if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Authorization header required" {
				t.Errorf("unexpected error message %q", msg)
			}
		})
	}
}

func TestAddToCartCreatesLineAndCartReportsTotals(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalogue(t, db)
	token := tokenFor(t, 1)

	status, raw := request(t, app, http.MethodPost, "/api/v1/cart", addBody(seed.dressID, seed.dressSizeID, 2), token)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}
	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Item added to cart" {
		t.Errorf("unexpected message %q", msg)
	}
	item := objectField(t, body, "item")
	if quantity := numberField(t, item, "quantity"); quantity != 2 {
		t.Errorf("expected quantity 2, got %v", quantity)
	}

	status, raw = request(t, app, http.MethodGet, "/api/v1/cart", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body = decodeJSON(t, raw)
	if count := numberField(t, body, "count"); count != 1 {
		t.Errorf("expected 1 cart line, got %v", count)
	}
	// 2 x 89.5.
	if subtotal := numberField(t, body, "subtotal"); subtotal != 179 {
		t.Errorf("expected subtotal 179, got %v", subtotal)
	}

	line := sliceField(t, body, "items")[0].(map[string]interface{})
	for field, want := range map[string]interface{}{
		"name":      "Silk Midi Dress",
		"image_url": "dress.png",
		"size":      "M",
		"color":     "Black",
	} {
		if got := line[field]; got != want {
			t.Errorf("expected %s %v, got %v", field, want, got)
		}
	}
	if price := numberField(t, line, "price"); price != 89.5 {
		t.Errorf("expected price 89.5, got %v", price)
	}
	if stock := numberField(t, line, "stock"); stock != 3 {
		t.Errorf("expected stock 3, got %v", stock)
	}
}

func TestAddToCartMergesRepeatAdditionsAndEnforcesStock(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalogue(t, db)
	token := tokenFor(t, 2)

	status, raw := request(t, app, http.MethodPost, "/api/v1/cart", addBody(seed.dressID, seed.dressSizeID, 1), token)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodPost, "/api/v1/cart", addBody(seed.dressID, seed.dressSizeID, 2), token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK when merging, got %d: %s", status, raw)
	}
	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Cart updated" {
		t.Errorf("unexpected message %q", msg)
	}
	if quantity := numberField(t, objectField(t, body, "item"), "quantity"); quantity != 3 {
		t.Errorf("expected merged quantity 3, got %v", quantity)
	}

	// One more unit would exceed the stock of the size variant.
	status, raw = request(t, app, http.MethodPost, "/api/v1/cart", addBody(seed.dressID, seed.dressSizeID, 1), token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Not enough stock available" {
		t.Errorf("unexpected error message %q", msg)
	}

	var lines []models.CartItem
	if err := db.Where("user_id = ?", 2).Find(&lines).Error; err != nil {
		t.Fatalf("loading cart: %v", err)
	}
	if len(lines) != 1 || lines[0].Quantity != 3 {
		t.Errorf("expected a single line with quantity 3, got %+v", lines)
	}
}

func TestAddToCartRejectsInvalidPayloadAndVariants(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalogue(t, db)
	token := tokenFor(t, 3)

	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "unknown size",
			body:        addBody(seed.dressID, 9999, 1),
			wantStatus:  fiber.StatusNotFound,
			wantMessage: "Size not found",
		},
		{
			name:        "size belongs to another product",
			body:        addBody(seed.dressID, seed.coatSizeID, 1),
			wantStatus:  fiber.StatusBadRequest,
			wantMessage: "Size does not belong to this product",
		},
		{
			name:        "size has no stock",
			body:        addBody(seed.dressID, seed.soldOutID, 1),
			wantStatus:  fiber.StatusBadRequest,
			wantMessage: "Not enough stock available",
		},
		{
			name:        "quantity exceeds stock",
			body:        addBody(seed.coatID, seed.coatSizeID, 3),
			wantStatus:  fiber.StatusBadRequest,
			wantMessage: "Not enough stock available",
		},
		{
			name:        "missing product id",
			body:        addBody(0, seed.dressSizeID, 1),
			wantStatus:  fiber.StatusBadRequest,
			wantMessage: "required",
		},
		{
			name:        "zero quantity",
			body:        addBody(seed.dressID, seed.dressSizeID, 0),
			wantStatus:  fiber.StatusBadRequest,
			wantMessage: "quantity",
		},
		{
			name:        "malformed json",
			body:        `{"product_id":`,
			wantStatus:  fiber.StatusBadRequest,
			wantMessage: "Invalid request body",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, "/api/v1/cart", tc.body, token)
			if status != tc.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tc.wantStatus, status, raw)
			}
			if msg := stringField(t, decodeJSON(t, raw), "error"); !strings.Contains(msg, tc.wantMessage) {
				t.Errorf("expected error to contain %q, got %q", tc.wantMessage, msg)
			}
		})
	}

	var lines []models.CartItem
	if err := db.Where("user_id = ?", 3).Find(&lines).Error; err != nil {
		t.Fatalf("loading cart: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected no cart lines after failed writes, got %+v", lines)
	}
}

func TestUpdateCartItemChangesQuantityAndChecksOwnership(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalogue(t, db)

	owner := tokenFor(t, 4)
	other := tokenFor(t, 5)

	status, raw := request(t, app, http.MethodPost, "/api/v1/cart", addBody(seed.dressID, seed.dressSizeID, 1), owner)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}

	var line models.CartItem
	if err := db.Where("user_id = ?", 4).First(&line).Error; err != nil {
		t.Fatalf("loading cart line: %v", err)
	}
	path := fmt.Sprintf("/api/v1/cart/%d", line.ID)

	status, raw = request(t, app, http.MethodPut, path, `{"quantity":2}`, owner)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Cart item updated" {
		t.Errorf("unexpected message %q", msg)
	}
	if quantity := numberField(t, objectField(t, body, "item"), "quantity"); quantity != 2 {
		t.Errorf("expected quantity 2, got %v", quantity)
	}

	// Exceeding the stock of the variant is rejected.
	status, raw = request(t, app, http.MethodPut, path, `{"quantity":4}`, owner)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Not enough stock available" {
		t.Errorf("unexpected error message %q", msg)
	}

	// Another user's line is invisible.
	status, raw = request(t, app, http.MethodPut, path, `{"quantity":1}`, other)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodPut, "/api/v1/cart/9999", `{"quantity":1}`, owner)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodPut, "/api/v1/cart/abc", `{"quantity":1}`, owner)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Invalid cart item ID" {
		t.Errorf("unexpected error message %q", msg)
	}

	if err := db.First(&line, line.ID).Error; err != nil {
		t.Fatalf("reloading cart line: %v", err)
	}
	if line.Quantity != 2 {
		t.Errorf("expected the quantity to stay 2, got %d", line.Quantity)
	}
}

func TestRemoveFromCartDeletesOnlyOwnedLines(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalogue(t, db)

	owner := tokenFor(t, 6)
	other := tokenFor(t, 7)

	for _, token := range []string{owner, other} {
		status, raw := request(t, app, http.MethodPost, "/api/v1/cart", addBody(seed.coatID, seed.coatSizeID, 1), token)
		if status != fiber.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", status, raw)
		}
	}

	var ownerLine models.CartItem
	if err := db.Where("user_id = ?", 6).First(&ownerLine).Error; err != nil {
		t.Fatalf("loading cart line: %v", err)
	}

	// The other user cannot delete a line that is not theirs.
	status, raw := request(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/cart/%d", ownerLine.ID), "", other)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/cart/%d", ownerLine.ID), "", owner)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "Item removed from cart" {
		t.Errorf("unexpected message %q", msg)
	}

	// Deleting again reports the missing line.
	status, raw = request(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/cart/%d", ownerLine.ID), "", owner)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}

	var remaining int64
	if err := db.Model(&models.CartItem{}).Where("user_id = ?", 6).Count(&remaining).Error; err != nil {
		t.Fatalf("counting cart lines: %v", err)
	}
	if remaining != 0 {
		t.Errorf("expected the owner cart to be empty, got %d lines", remaining)
	}

	status, raw = request(t, app, http.MethodDelete, "/api/v1/cart/abc", "", owner)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Invalid cart item ID" {
		t.Errorf("unexpected error message %q", msg)
	}
}

func TestClearCartEmptiesOnlyTheCallersCart(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalogue(t, db)

	owner := tokenFor(t, 8)
	other := tokenFor(t, 9)

	payloads := []string{
		addBody(seed.dressID, seed.dressSizeID, 1),
		addBody(seed.coatID, seed.coatSizeID, 1),
	}
	for _, body := range payloads {
		if status, raw := request(t, app, http.MethodPost, "/api/v1/cart", body, owner); status != fiber.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", status, raw)
		}
	}
	if status, raw := request(t, app, http.MethodPost, "/api/v1/cart", addBody(seed.coatID, seed.coatSizeID, 1), other); status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}

	status, raw := request(t, app, http.MethodDelete, "/api/v1/cart", "", owner)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "Cart cleared" {
		t.Errorf("unexpected message %q", msg)
	}

	countFor := func(userID uint) int64 {
		var count int64
		if err := db.Model(&models.CartItem{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
			t.Fatalf("counting cart lines: %v", err)
		}
		return count
	}
	if got := countFor(8); got != 0 {
		t.Errorf("expected the owner cart to be empty, got %d lines", got)
	}
	if got := countFor(9); got != 1 {
		t.Errorf("expected the other cart to be untouched, got %d lines", got)
	}

	// Clearing an already empty cart stays successful.
	if status, raw = request(t, app, http.MethodDelete, "/api/v1/cart", "", owner); status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
}

func TestGetWishlistReportsStockAvailability(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalogue(t, db)

	// A product without any size variant can never be in stock.
	dryProduct := testProduct{Name: "Sold Out Blazer", ImageURL: "blazer.png", Price: 120}
	if err := db.Create(&dryProduct).Error; err != nil {
		t.Fatalf("creating product: %v", err)
	}

	token := tokenFor(t, 10)
	for _, productID := range []uint{seed.dressID, dryProduct.ID} {
		status, raw := request(t, app, http.MethodPost, fmt.Sprintf("/api/v1/wishlist/%d", productID), "", token)
		if status != fiber.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", status, raw)
		}
	}

	status, raw := request(t, app, http.MethodGet, "/api/v1/wishlist", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if count := numberField(t, body, "count"); count != 2 {
		t.Errorf("expected 2 wishlist entries, got %v", count)
	}

	items := sliceField(t, body, "items")
	byName := make(map[string]map[string]interface{}, len(items))
	for _, entry := range items {
		item := entry.(map[string]interface{})
		byName[fmt.Sprint(item["name"])] = item
	}

	dress, ok := byName["Silk Midi Dress"]
	if !ok {
		t.Fatalf("expected the dress in the wishlist, got %s", raw)
	}
	if inStock, _ := dress["in_stock"].(bool); !inStock {
		t.Errorf("expected the dress to be in stock, got %v", dress)
	}
	if price := numberField(t, dress, "price"); price != 89.5 {
		t.Errorf("expected price 89.5, got %v", price)
	}

	blazer, ok := byName["Sold Out Blazer"]
	if !ok {
		t.Fatalf("expected the blazer in the wishlist, got %s", raw)
	}
	if inStock, _ := blazer["in_stock"].(bool); inStock {
		t.Errorf("expected the blazer to be out of stock, got %v", blazer)
	}
}

func TestToggleWishlistItemAddsThenRemoves(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalogue(t, db)

	token := tokenFor(t, 11)
	path := fmt.Sprintf("/api/v1/wishlist/%d", seed.dressID)

	status, raw := request(t, app, http.MethodPost, path, "", token)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}
	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Added to wishlist" {
		t.Errorf("unexpected message %q", msg)
	}
	if wishlisted, _ := body["wishlisted"].(bool); !wishlisted {
		t.Errorf("expected wishlisted true, got %v", body)
	}
	if productID := numberField(t, objectField(t, body, "item"), "product_id"); productID != float64(seed.dressID) {
		t.Errorf("expected product %d, got %v", seed.dressID, productID)
	}

	// Toggling again removes the entry.
	status, raw = request(t, app, http.MethodPost, path, "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body = decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Removed from wishlist" {
		t.Errorf("unexpected message %q", msg)
	}
	if wishlisted, _ := body["wishlisted"].(bool); wishlisted {
		t.Errorf("expected wishlisted false, got %v", body)
	}

	var remaining int64
	if err := db.Model(&models.WishlistItem{}).Where("user_id = ?", 11).Count(&remaining).Error; err != nil {
		t.Fatalf("counting wishlist entries: %v", err)
	}
	if remaining != 0 {
		t.Errorf("expected an empty wishlist, got %d entries", remaining)
	}

	// Unknown and malformed product IDs are rejected before any write.
	status, raw = request(t, app, http.MethodPost, "/api/v1/wishlist/4242", "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Product not found" {
		t.Errorf("unexpected error message %q", msg)
	}

	status, raw = request(t, app, http.MethodPost, "/api/v1/wishlist/abc", "", token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Invalid product ID" {
		t.Errorf("unexpected error message %q", msg)
	}
}
