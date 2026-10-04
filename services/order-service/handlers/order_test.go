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
	"time"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/order-service/database"
	"github.com/adibfahimi/moda-style/services/order-service/handlers"
	"github.com/adibfahimi/moda-style/services/order-service/models"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testJWTSecret is the signing key every test in this package uses. TestMain
// installs it before any test runs because common resolves JWT_SECRET lazily.
const testJWTSecret = "order-service-test-secret"

// dbCounter hands every test its own in-memory database.
var dbCounter atomic.Uint64

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

// testProduct mirrors the products table of the product service, which this
// service reads through the shared database during checkout.
type testProduct struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Price float64
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

// testCartItem mirrors the cart_lines table of the cart service.
type testCartItem struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	ProductID uint
	SizeID    uint
	Quantity  int
}

// TableName pins the test model onto the shared production table name.
func (testCartItem) TableName() string { return "cart_items" }

// newTestDB returns an isolated in-memory SQLite database with the order schema
// plus the shared cart/product/size tables migrated, and points database.DB at
// it.
//
// SQLite keeps the handler tests self-contained; the SQL used here (joins,
// UPDATE ... WHERE, soft deletes) behaves identically on PostgreSQL.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:order_test_%d?mode=memory&cache=shared", dbCounter.Add(1))
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

	if err := db.AutoMigrate(
		&models.Order{}, &models.OrderItem{}, &models.PaymentTransaction{},
		&testProduct{}, &testSize{}, &testCartItem{},
	); err != nil {
		t.Fatalf("migrating order schema: %v", err)
	}

	database.DB = db
	return db
}

// newTestApp mounts the same routes as services/order-service/main.go.
func newTestApp() *fiber.App {
	app := fiber.New()

	api := app.Group("/api/v1")
	api.Post("/orders", common.RequireAuth, handlers.CreateOrder)
	api.Post("/orders/:id/pay", common.RequireAuth, handlers.ProcessPayment)
	api.Get("/orders/my-orders", common.RequireAuth, handlers.GetMyOrders)
	api.Get("/orders/:id", common.RequireAuth, handlers.GetOrderByID)
	api.Post("/orders/:id/cancel", common.RequireAuth, handlers.CancelOrder)

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

// decodeJSONArray unmarshals a JSON array body, as returned by the order list.
func decodeJSONArray(t *testing.T, raw []byte) []interface{} {
	t.Helper()

	var decoded []interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("response is not a valid JSON array (%q): %v", string(raw), err)
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

// seededShop is the catalogue fixture the checkout tests order from.
type seededShop struct {
	dressID     uint
	dressSizeID uint // M/Black, three in stock, 89.50 each
	coatID      uint
	coatSizeID  uint // XL/Grey, two in stock, 180.00 each
}

// seedShop inserts two products with one size variant each.
func seedShop(t *testing.T, db *gorm.DB) seededShop {
	t.Helper()

	dress := testProduct{Name: "Silk Midi Dress", Price: 89.5}
	coat := testProduct{Name: "Wool Coat", Price: 180}
	for _, product := range []*testProduct{&dress, &coat} {
		if err := db.Create(product).Error; err != nil {
			t.Fatalf("creating product: %v", err)
		}
	}

	dressSize := testSize{ProductID: dress.ID, Size: "M", Color: "Black", Stock: 3}
	coatSize := testSize{ProductID: coat.ID, Size: "XL", Color: "Grey", Stock: 2}
	for _, size := range []*testSize{&dressSize, &coatSize} {
		if err := db.Create(size).Error; err != nil {
			t.Fatalf("creating size: %v", err)
		}
	}

	return seededShop{
		dressID:     dress.ID,
		dressSizeID: dressSize.ID,
		coatID:      coat.ID,
		coatSizeID:  coatSize.ID,
	}
}

// seedCartLine puts one line into the shared cart_items table, mimicking what
// the cart service would have written before checkout.
func seedCartLine(t *testing.T, db *gorm.DB, userID, sizeID, quantity int) testCartItem {
	t.Helper()

	var size testSize
	if err := db.First(&size, sizeID).Error; err != nil {
		t.Fatalf("loading size %d: %v", sizeID, err)
	}

	line := testCartItem{UserID: uint(userID), ProductID: size.ProductID, SizeID: uint(sizeID), Quantity: quantity}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("creating cart line: %v", err)
	}
	return line
}

// createOrder performs a checkout through the API and fails the test when it
// does not return 201, so the payment and cancellation tests start from a real
// order row.
func createOrder(t *testing.T, app *fiber.App, token string) models.Order {
	t.Helper()

	status, raw := request(t, app, http.MethodPost, "/api/v1/orders",
		`{"shipping_address":"12 Rue de la Mode, Paris"}`, token)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}

	var order models.Order
	if err := json.Unmarshal(mustJSON(t, objectField(t, decodeJSON(t, raw), "order")), &order); err != nil {
		t.Fatalf("decoding order: %v", err)
	}
	return order
}

// mustJSON re-encodes a decoded JSON object so it can be unmarshalled into a
// typed struct.
func mustJSON(t *testing.T, object map[string]interface{}) []byte {
	t.Helper()

	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("re-encoding JSON object: %v", err)
	}
	return raw
}

func TestOrderRoutesRequireAuthentication(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/orders"},
		{http.MethodPost, "/api/v1/orders/1/pay"},
		{http.MethodGet, "/api/v1/orders/my-orders"},
		{http.MethodGet, "/api/v1/orders/1"},
		{http.MethodPost, "/api/v1/orders/1/cancel"},
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

func TestCreateOrderValidatesRequest(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seedShop(t, db)
	token := tokenFor(t, 1)

	tests := []struct {
		name        string
		body        string
		seedCart    bool
		wantMessage string
	}{
		{
			name:        "malformed body",
			body:        `{"shipping_address":`,
			wantMessage: "Invalid request body",
		},
		{
			name:        "missing shipping address",
			body:        `{}`,
			wantMessage: "Shipping address is required",
		},
		{
			name:        "empty cart",
			body:        `{"shipping_address":"12 Rue de la Mode, Paris"}`,
			wantMessage: "Cart is empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, "/api/v1/orders", tc.body, token)
			if status != fiber.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
			}
			if msg := stringField(t, decodeJSON(t, raw), "error"); msg != tc.wantMessage {
				t.Errorf("expected %q, got %q", tc.wantMessage, msg)
			}
		})
	}
}

func TestCreateOrderBuildsItemsFromTheCart(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedShop(t, db)
	token := tokenFor(t, 2)

	seedCartLine(t, db, 2, int(seed.dressSizeID), 2)
	seedCartLine(t, db, 2, int(seed.coatSizeID), 1)

	status, raw := request(t, app, http.MethodPost, "/api/v1/orders",
		`{"shipping_address":"12 Rue de la Mode, Paris","notes":"Leave at the door"}`, token)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Order created successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	order := objectField(t, body, "order")
	// 2 x 89.5 + 1 x 180.
	if total := numberField(t, order, "total_amount"); total != 359 {
		t.Errorf("expected total 359, got %v", total)
	}
	if orderNumber := stringField(t, order, "order_number"); !strings.HasPrefix(orderNumber, "ORD-") {
		t.Errorf("expected a generated order number, got %q", orderNumber)
	}
	if status := stringField(t, order, "status"); status != "pending" {
		t.Errorf("expected a pending order, got %q", status)
	}
	if paymentStatus := stringField(t, order, "payment_status"); paymentStatus != "pending" {
		t.Errorf("expected pending payment, got %q", paymentStatus)
	}
	// The payment method defaults to card when the client omits it.
	if method := stringField(t, order, "payment_method"); method != "card" {
		t.Errorf("expected the card default, got %q", method)
	}

	items := sliceField(t, order, "items")
	if len(items) != 2 {
		t.Fatalf("expected 2 order items, got %d: %s", len(items), raw)
	}

	byName := make(map[string]map[string]interface{}, len(items))
	for _, entry := range items {
		item := entry.(map[string]interface{})
		byName[fmt.Sprint(item["product_name"])] = item
	}

	dress, ok := byName["Silk Midi Dress"]
	if !ok {
		t.Fatalf("expected the dress line, got %s", raw)
	}
	if size := stringField(t, dress, "size"); size != "M" {
		t.Errorf("expected size M, got %q", size)
	}
	if color := stringField(t, dress, "color"); color != "Black" {
		t.Errorf("expected colour Black, got %q", color)
	}
	if subtotal := numberField(t, dress, "subtotal"); subtotal != 179 {
		t.Errorf("expected subtotal 179, got %v", subtotal)
	}

	// The order and its items are persisted, but the cart stays untouched until
	// the payment succeeds.
	var stored models.Order
	if err := db.Preload("Items").Where("user_id = ?", 2).First(&stored).Error; err != nil {
		t.Fatalf("loading order: %v", err)
	}
	if len(stored.Items) != 2 {
		t.Errorf("expected 2 persisted items, got %d", len(stored.Items))
	}

	var cartLines int64
	if err := db.Model(&testCartItem{}).Where("user_id = ?", 2).Count(&cartLines).Error; err != nil {
		t.Fatalf("counting cart lines: %v", err)
	}
	if cartLines != 2 {
		t.Errorf("expected the cart to be untouched before payment, got %d lines", cartLines)
	}
}

func TestCreateOrderRejectsUnknownProductAndInsufficientStock(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedShop(t, db)
	token := tokenFor(t, 3)

	// A cart line pointing at a size that no longer exists in the catalogue.
	if err := db.Create(&testCartItem{UserID: 3, ProductID: seed.dressID, SizeID: 7777, Quantity: 1}).Error; err != nil {
		t.Fatalf("creating cart line: %v", err)
	}

	status, raw := request(t, app, http.MethodPost, "/api/v1/orders",
		`{"shipping_address":"12 Rue de la Mode, Paris"}`, token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); !strings.Contains(msg, "not found") {
		t.Errorf("expected a missing product error, got %q", msg)
	}

	// Replace the broken line with one that asks for more than the stock.
	if err := db.Where("user_id = ?", 3).Delete(&testCartItem{}).Error; err != nil {
		t.Fatalf("clearing cart: %v", err)
	}
	seedCartLine(t, db, 3, int(seed.dressSizeID), 5)

	status, raw = request(t, app, http.MethodPost, "/api/v1/orders",
		`{"shipping_address":"12 Rue de la Mode, Paris"}`, token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	msg := stringField(t, decodeJSON(t, raw), "error")
	if !strings.Contains(msg, "Insufficient stock for Silk Midi Dress") {
		t.Errorf("expected an insufficient stock error, got %q", msg)
	}

	var orders int64
	if err := db.Model(&models.Order{}).Count(&orders).Error; err != nil {
		t.Fatalf("counting orders: %v", err)
	}
	if orders != 0 {
		t.Errorf("expected no order to be written, got %d", orders)
	}
}

func TestProcessPaymentSucceedsAndConsumesTheCart(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedShop(t, db)
	token := tokenFor(t, 4)

	seedCartLine(t, db, 4, int(seed.dressSizeID), 2)
	order := createOrder(t, app, token)

	status, raw := request(t, app, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/pay", order.ID),
		`{"payment_intent_id":"pi_test_success"}`, token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if success, _ := body["success"].(bool); !success {
		t.Errorf("expected success true, got %v", body)
	}
	if msg := stringField(t, body, "message"); msg != "Payment processed successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	paid := objectField(t, body, "order")
	if paymentStatus := stringField(t, paid, "payment_status"); paymentStatus != "paid" {
		t.Errorf("expected the order to be paid, got %q", paymentStatus)
	}
	if orderStatus := stringField(t, paid, "status"); orderStatus != "processing" {
		t.Errorf("expected the order to be processing, got %q", orderStatus)
	}
	if intent := stringField(t, paid, "payment_intent_id"); intent != "pi_test_success" {
		t.Errorf("expected the payment intent to be stored, got %q", intent)
	}

	// Stock is decremented by the ordered quantity: 3 - 2.
	var size testSize
	if err := db.First(&size, seed.dressSizeID).Error; err != nil {
		t.Fatalf("loading size: %v", err)
	}
	if size.Stock != 1 {
		t.Errorf("expected remaining stock 1, got %d", size.Stock)
	}

	// The cart is emptied and a succeeded transaction is recorded.
	var cartLines int64
	if err := db.Model(&testCartItem{}).Where("user_id = ?", 4).Count(&cartLines).Error; err != nil {
		t.Fatalf("counting cart lines: %v", err)
	}
	if cartLines != 0 {
		t.Errorf("expected the cart to be emptied, got %d lines", cartLines)
	}

	var transaction models.PaymentTransaction
	if err := db.Where("order_id = ?", order.ID).First(&transaction).Error; err != nil {
		t.Fatalf("loading transaction: %v", err)
	}
	if transaction.Status != "succeeded" {
		t.Errorf("expected a succeeded transaction, got %q", transaction.Status)
	}
	if transaction.Amount != order.TotalAmount {
		t.Errorf("expected the transaction amount %v, got %v", order.TotalAmount, transaction.Amount)
	}
}

func TestProcessPaymentDeclinesAndKeepsStockAndCart(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedShop(t, db)
	token := tokenFor(t, 5)

	seedCartLine(t, db, 5, int(seed.dressSizeID), 1)
	order := createOrder(t, app, token)

	// The Stripe decline suffix makes the simulated processor fail.
	status, raw := request(t, app, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/pay", order.ID),
		`{"payment_intent_id":"pi_test_0000000002"}`, token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if success, _ := body["success"].(bool); success {
		t.Errorf("expected success false, got %v", body)
	}
	if msg := stringField(t, body, "error"); msg != "Payment failed" {
		t.Errorf("unexpected error message %q", msg)
	}
	if paymentStatus := stringField(t, objectField(t, body, "order"), "payment_status"); paymentStatus != "failed" {
		t.Errorf("expected the order to be marked failed, got %q", paymentStatus)
	}

	// Stock and cart are untouched, and the attempt is recorded as failed.
	var size testSize
	if err := db.First(&size, seed.dressSizeID).Error; err != nil {
		t.Fatalf("loading size: %v", err)
	}
	if size.Stock != 3 {
		t.Errorf("expected the stock to stay 3, got %d", size.Stock)
	}

	var cartLines int64
	if err := db.Model(&testCartItem{}).Where("user_id = ?", 5).Count(&cartLines).Error; err != nil {
		t.Fatalf("counting cart lines: %v", err)
	}
	if cartLines != 1 {
		t.Errorf("expected the cart to be kept after a decline, got %d lines", cartLines)
	}

	var transaction models.PaymentTransaction
	if err := db.Where("order_id = ?", order.ID).First(&transaction).Error; err != nil {
		t.Fatalf("loading transaction: %v", err)
	}
	if transaction.Status != "failed" || transaction.ErrorMessage == "" {
		t.Errorf("expected a failed transaction with a message, got %+v", transaction)
	}
}

func TestProcessPaymentValidatesInputAndOrderOwnership(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedShop(t, db)

	owner := tokenFor(t, 6)
	other := tokenFor(t, 7)

	seedCartLine(t, db, 6, int(seed.dressSizeID), 1)
	order := createOrder(t, app, owner)
	payPath := fmt.Sprintf("/api/v1/orders/%d/pay", order.ID)

	status, raw := request(t, app, http.MethodPost, payPath, `{"payment_intent_id":`, owner)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Invalid request body" {
		t.Errorf("unexpected error message %q", msg)
	}

	status, raw = request(t, app, http.MethodPost, payPath, `{}`, owner)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Payment intent ID is required" {
		t.Errorf("unexpected error message %q", msg)
	}

	status, raw = request(t, app, http.MethodPost, payPath, `{"payment_intent_id":"pi_test_success"}`, other)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found for another user's order, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodPost, "/api/v1/orders/9999/pay", `{"payment_intent_id":"pi_test_success"}`, owner)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}

	// A paid order cannot be charged twice.
	if status, raw = request(t, app, http.MethodPost, payPath, `{"payment_intent_id":"pi_test_success"}`, owner); status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	status, raw = request(t, app, http.MethodPost, payPath, `{"payment_intent_id":"pi_test_success"}`, owner)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Order already paid" {
		t.Errorf("unexpected error message %q", msg)
	}
}

func TestGetMyOrdersReturnsOnlyTheCallersOrders(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedShop(t, db)

	first := tokenFor(t, 10)
	second := tokenFor(t, 11)

	seedCartLine(t, db, 10, int(seed.dressSizeID), 1)
	older := createOrder(t, app, first)

	// A pause keeps the two created_at values distinct so the newest-first
	// ordering can be asserted reliably.
	time.Sleep(10 * time.Millisecond)
	seedCartLine(t, db, 10, int(seed.coatSizeID), 1)
	newer := createOrder(t, app, first)

	seedCartLine(t, db, 11, int(seed.dressSizeID), 1)
	foreign := createOrder(t, app, second)

	status, raw := request(t, app, http.MethodGet, "/api/v1/orders/my-orders", "", first)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	orders := decodeJSONArray(t, raw)
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders for the caller, got %d: %s", len(orders), raw)
	}

	// Newest first.
	if got := numberField(t, orders[0].(map[string]interface{}), "id"); uint(got) != newer.ID {
		t.Errorf("expected the newest order first, got id %v", got)
	}
	if got := numberField(t, orders[1].(map[string]interface{}), "id"); uint(got) != older.ID {
		t.Errorf("expected the older order second, got id %v", got)
	}

	for _, entry := range orders {
		order := entry.(map[string]interface{})
		if numberField(t, order, "id") == float64(foreign.ID) {
			t.Errorf("another user's order leaked into the response: %s", raw)
		}
		if items := sliceField(t, order, "items"); len(items) == 0 {
			t.Errorf("expected the items to be preloaded, got %v", order)
		}
	}
}

func TestGetOrderByIDScopesToTheCaller(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedShop(t, db)

	owner := tokenFor(t, 12)
	other := tokenFor(t, 13)

	seedCartLine(t, db, 12, int(seed.dressSizeID), 2)
	order := createOrder(t, app, owner)
	path := fmt.Sprintf("/api/v1/orders/%d", order.ID)

	status, raw := request(t, app, http.MethodGet, path, "", owner)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if id := numberField(t, body, "id"); uint(id) != order.ID {
		t.Errorf("expected order %d, got %v", order.ID, id)
	}
	items := sliceField(t, body, "items")
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d: %s", len(items), raw)
	}
	if name := stringField(t, items[0].(map[string]interface{}), "product_name"); name != "Silk Midi Dress" {
		t.Errorf("expected the dress item, got %q", name)
	}

	for _, tc := range []struct {
		name   string
		path   string
		token  string
		status int
	}{
		{"another user's order", path, other, fiber.StatusNotFound},
		{"unknown order", "/api/v1/orders/4242", owner, fiber.StatusNotFound},
		{"non numeric id", "/api/v1/orders/not-a-number", owner, fiber.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, tc.path, "", tc.token)
			if status != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, status, raw)
			}
		})
	}
}

func TestCancelOrderCancelsPendingAndRefundsPaid(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedShop(t, db)

	token := tokenFor(t, 14)
	other := tokenFor(t, 15)

	seedCartLine(t, db, 14, int(seed.dressSizeID), 1)
	pending := createOrder(t, app, token)
	pendingPath := fmt.Sprintf("/api/v1/orders/%d/cancel", pending.ID)

	status, raw := request(t, app, http.MethodPost, pendingPath, "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Order cancelled successfully" {
		t.Errorf("unexpected message %q", msg)
	}
	if orderStatus := stringField(t, objectField(t, body, "order"), "status"); orderStatus != "cancelled" {
		t.Errorf("expected the order to be cancelled, got %q", orderStatus)
	}

	// A cancelled order cannot be cancelled again.
	status, raw = request(t, app, http.MethodPost, pendingPath, "", token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); !strings.Contains(msg, "Cannot cancel order") {
		t.Errorf("unexpected error message %q", msg)
	}

	// Paying for the coat consumes its stock; cancelling the paid order refunds
	// it and gives the units back.
	seedCartLine(t, db, 14, int(seed.coatSizeID), 2)
	paid := createOrder(t, app, token)
	paidPath := fmt.Sprintf("/api/v1/orders/%d/cancel", paid.ID)
	if status, raw := request(t, app, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/pay", paid.ID),
		`{"payment_intent_id":"pi_test_success"}`, token); status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	var coatSize testSize
	if err := db.First(&coatSize, seed.coatSizeID).Error; err != nil {
		t.Fatalf("loading size: %v", err)
	}
	if coatSize.Stock != 0 {
		t.Fatalf("expected the coat to be sold out, got %d", coatSize.Stock)
	}

	status, raw = request(t, app, http.MethodPost, paidPath, "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	refunded := objectField(t, decodeJSON(t, raw), "order")
	if orderStatus := stringField(t, refunded, "status"); orderStatus != "cancelled" {
		t.Errorf("expected the order to be cancelled, got %q", orderStatus)
	}
	if paymentStatus := stringField(t, refunded, "payment_status"); paymentStatus != "refunded" {
		t.Errorf("expected the payment to be refunded, got %q", paymentStatus)
	}

	if err := db.First(&coatSize, seed.coatSizeID).Error; err != nil {
		t.Fatalf("loading size: %v", err)
	}
	if coatSize.Stock != 2 {
		t.Errorf("expected the stock to be restored to 2, got %d", coatSize.Stock)
	}

	// Ownership and existence checks.
	for _, tc := range []struct {
		name   string
		path   string
		token  string
		status int
	}{
		{"another user's order", paidPath, other, fiber.StatusNotFound},
		{"unknown order", "/api/v1/orders/9999/cancel", token, fiber.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, tc.path, "", tc.token)
			if status != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, status, raw)
			}
		})
	}
}
