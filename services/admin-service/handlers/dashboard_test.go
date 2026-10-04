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
	"github.com/adibfahimi/moda-style/services/admin-service/database"
	"github.com/adibfahimi/moda-style/services/admin-service/handlers"
	"github.com/adibfahimi/moda-style/services/admin-service/models"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testJWTSecret is the signing key every test in this package uses. TestMain
// installs it before any test runs because common resolves JWT_SECRET lazily.
const testJWTSecret = "admin-service-test-secret"

// dbCounter hands every test its own in-memory database.
var dbCounter atomic.Uint64

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

// testUser mirrors the users table that the auth service owns but this service
// reads and mutates through the shared database.
type testUser struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	Email     string
	Role      string
	Banned    bool
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName pins the test model onto the shared production table name.
func (testUser) TableName() string { return "users" }

// testProduct mirrors the products table of the product service.
type testProduct struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Description string
	CategoryID  uint
	Price       float64
	ImageURL    string
	CreatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName pins the test model onto the shared production table name.
func (testProduct) TableName() string { return "products" }

// testCategory mirrors the categories table of the product service.
type testCategory struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	Slug      string
	ParentID  *uint
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName pins the test model onto the shared production table name.
func (testCategory) TableName() string { return "categories" }

// testReview mirrors the reviews table of the product service.
type testReview struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	ProductID uint
	UserName  string
	Rating    int
	Comment   string
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName pins the test model onto the shared production table name.
func (testReview) TableName() string { return "reviews" }

// testSize mirrors the sizes table of the product service. The dashboard sums
// its stock to find products that need restocking.
type testSize struct {
	ID        uint `gorm:"primaryKey"`
	ProductID uint
	Size      string
	Color     string
	Stock     int
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName pins the test model onto the shared production table name.
func (testSize) TableName() string { return "sizes" }

// testOrder mirrors the orders table owned by the order service.
type testOrder struct {
	ID              uint `gorm:"primaryKey"`
	UserID          uint
	OrderNumber     string
	Status          string
	TotalAmount     float64
	ShippingAddress string
	PaymentMethod   string
	PaymentStatus   string
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

// TableName pins the test model onto the shared production table name.
func (testOrder) TableName() string { return "orders" }

// testOrderItem mirrors the order_items table owned by the order service.
type testOrderItem struct {
	ID          uint `gorm:"primaryKey"`
	OrderID     uint
	ProductID   uint
	ProductName string
	Quantity    int
	Price       float64
	Subtotal    float64
	CreatedAt   time.Time
}

// TableName pins the test model onto the shared production table name.
func (testOrderItem) TableName() string { return "order_items" }

// testWishlistItem mirrors the wishlist_items table of the cart service.
type testWishlistItem struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	ProductID uint
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName pins the test model onto the shared production table name.
func (testWishlistItem) TableName() string { return "wishlist_items" }

// newTestDB returns an isolated in-memory SQLite database with the admin schema
// plus every table the dashboard reads, and points database.DB at it.
//
// SQLite keeps the handler tests self-contained; the SQL used here (JOINs,
// COUNT, AVG, LIKE and string concatenation) behaves identically on PostgreSQL.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:admin_test_%d?mode=memory&cache=shared", dbCounter.Add(1))
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
		&models.ActivityLog{},
		&testUser{},
		&testProduct{},
		&testCategory{},
		&testReview{},
		&testSize{},
		&testWishlistItem{},
		&testOrder{},
		&testOrderItem{},
	); err != nil {
		t.Fatalf("migrating admin schema: %v", err)
	}

	database.DB = db
	return db
}

// newTestApp mounts the same admin routes as services/admin-service/main.go.
//
// Every route sits behind the production auth chain, so the tests exercise the
// real 401/403 behaviour of common.RequireAuth + common.RequireAdmin.
func newTestApp() *fiber.App {
	app := fiber.New()

	admin := app.Group("/api/v1/admin", common.RequireAuth, common.RequireAdmin)

	admin.Get("/dashboard/stats", handlers.GetDashboardStats)
	admin.Get("/dashboard/activity", handlers.GetRecentActivity)
	admin.Get("/dashboard/logs", handlers.GetActivityLogs)

	admin.Get("/users", handlers.GetUsers)
	admin.Get("/users/analytics", handlers.GetUserAnalytics)
	admin.Get("/users/:id", handlers.GetUserDetails)
	admin.Put("/users/:id", handlers.UpdateUser)
	admin.Delete("/users/:id", handlers.DeleteUser)
	admin.Post("/users/:id/ban", handlers.BanUser)
	admin.Post("/users/:id/unban", handlers.UnbanUser)

	admin.Get("/products", handlers.GetProductStats)
	admin.Get("/products/images", handlers.ListProductImages)
	admin.Post("/products/upload-image", handlers.UploadProductImage)
	admin.Post("/products", handlers.CreateProduct)
	admin.Put("/products/:id", handlers.UpdateProduct)
	admin.Delete("/products/:id", handlers.DeleteProduct)

	admin.Get("/products/:id/sizes", handlers.GetProductSizes)
	admin.Post("/products/:id/sizes", handlers.AddProductSize)
	admin.Put("/products/:id/sizes/:sizeId", handlers.UpdateProductSize)
	admin.Delete("/products/:id/sizes/:sizeId", handlers.DeleteProductSize)

	admin.Get("/categories", handlers.GetCategories)
	admin.Post("/categories", handlers.CreateCategory)
	admin.Put("/categories/:id", handlers.UpdateCategory)
	admin.Delete("/categories/:id", handlers.DeleteCategory)

	admin.Get("/orders", handlers.GetOrders)
	admin.Get("/orders/analytics", handlers.GetOrderAnalytics)
	admin.Get("/orders/:id", handlers.GetOrderDetails)
	admin.Put("/orders/:id", handlers.UpdateOrderStatus)
	admin.Delete("/orders/:id", handlers.DeleteOrder)

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

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("performing request: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	return resp.StatusCode, raw
}

// decodeJSON unmarshals a response body into a generic object.
func decodeJSON(t *testing.T, raw []byte) map[string]interface{} {
	t.Helper()

	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decoding JSON %q: %v", raw, err)
	}
	return body
}

// decodeJSONArray unmarshals a response body into a generic array.
func decodeJSONArray(t *testing.T, raw []byte) []interface{} {
	t.Helper()

	var body []interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decoding JSON array %q: %v", raw, err)
	}
	return body
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

// sliceField returns the named JSON array, failing the test when it is missing.
func sliceField(t *testing.T, body map[string]interface{}, key string) []interface{} {
	t.Helper()

	value, ok := body[key].([]interface{})
	if !ok {
		t.Fatalf("field %q is missing or not an array in %v", key, body)
	}
	return value
}

// adminToken issues a token for an admin account.
func adminToken(t *testing.T, userID uint) string {
	t.Helper()
	return tokenFor(t, userID, "admin")
}

// tokenFor issues a valid access token with the given role so tests can reach
// the routes that sit behind common.RequireAuth + common.RequireAdmin.
func tokenFor(t *testing.T, userID uint, role string) string {
	t.Helper()

	token, err := common.GenerateToken(userID, fmt.Sprintf("user%d@example.com", userID), "Ada", role)
	if err != nil {
		t.Fatalf("generating token: %v", err)
	}
	return token
}

// TestEmptyCollectionsSerializeAsArrays guards the API contract the SPA relies
// on: list endpoints must answer with [] instead of null when there is nothing
// to show, so the frontend can map over the response without a null check.
func TestEmptyCollectionsSerializeAsArrays(t *testing.T) {
	newTestDB(t)
	app := newTestApp()
	token := adminToken(t, 1)

	for _, path := range []string{
		"/api/v1/admin/dashboard/activity",
		"/api/v1/admin/dashboard/logs",
		"/api/v1/admin/users",
	} {
		t.Run(path, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, path, "", token)
			if status != fiber.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", status, raw)
			}

			body := decodeJSON(t, raw)
			for _, key := range []string{"activities", "logs", "users"} {
				value, ok := body[key]
				if !ok {
					continue
				}
				if _, isArray := value.([]interface{}); !isArray {
					t.Errorf("expected %q to be an array, got %v (%s)", key, value, raw)
				}
			}
		})
	}
}

// seedUser inserts one row into the shared users table.
func seedUser(t *testing.T, db *gorm.DB, name, email, role string) testUser {
	t.Helper()

	user := testUser{Name: name, Email: email, Role: role}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("creating user: %v", err)
	}
	return user
}

// seedCatalogue inserts one category, one product and its sizes.
//
// The sizes are what the dashboard sums to detect low stock.
func seedCatalogue(t *testing.T, db *gorm.DB, name string, stock int) (testCategory, testProduct) {
	t.Helper()

	category := testCategory{Name: "Dresses"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("creating category: %v", err)
	}

	product := testProduct{Name: name, CategoryID: category.ID, Price: 89.5}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("creating product: %v", err)
	}

	size := testSize{ProductID: product.ID, Size: "M", Color: "Black", Stock: stock}
	if err := db.Create(&size).Error; err != nil {
		t.Fatalf("creating size: %v", err)
	}

	return category, product
}

// seedReview inserts one review for a product on behalf of a user.
func seedReview(t *testing.T, db *gorm.DB, user testUser, product testProduct, rating int) testReview {
	t.Helper()

	review := testReview{
		UserID:    user.ID,
		ProductID: product.ID,
		UserName:  user.Name,
		Rating:    rating,
		Comment:   "Lovely fabric",
	}
	if err := db.Create(&review).Error; err != nil {
		t.Fatalf("creating review: %v", err)
	}
	return review
}

// seedOrder inserts an order plus one line item and returns both.
//
// The paid orders record a payment_status of "paid" so the revenue analytics
// have something to sum.
func seedOrder(t *testing.T, db *gorm.DB, user testUser, number, status, paymentStatus string, total float64) (testOrder, testOrderItem) {
	t.Helper()

	order := testOrder{
		UserID:          user.ID,
		OrderNumber:     number,
		Status:          status,
		TotalAmount:     total,
		ShippingAddress: "12 Rue de la Paix, Paris",
		PaymentMethod:   "credit_card",
		PaymentStatus:   paymentStatus,
		Notes:           "",
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("creating order: %v", err)
	}

	item := testOrderItem{
		OrderID:     order.ID,
		ProductID:   1,
		ProductName: "Silk Midi Dress",
		Quantity:    1,
		Price:       total,
		Subtotal:    total,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("creating order item: %v", err)
	}

	return order, item
}

func TestDashboardRoutesRequireTheAdminRole(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	routes := []string{
		"/api/v1/admin/dashboard/stats",
		"/api/v1/admin/dashboard/activity",
		"/api/v1/admin/dashboard/logs",
	}

	for _, path := range routes {
		t.Run(path, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, path, "", "")
			if status != fiber.StatusUnauthorized {
				t.Fatalf("expected 401 without a token, got %d: %s", status, raw)
			}

			status, raw = request(t, app, http.MethodGet, path, "", tokenFor(t, 99, "user"))
			if status != fiber.StatusForbidden {
				t.Fatalf("expected 403 for a non-admin, got %d: %s", status, raw)
			}

			status, raw = request(t, app, http.MethodGet, path, "", adminToken(t, 100))
			if status != fiber.StatusOK {
				t.Fatalf("expected 200 for an admin, got %d: %s", status, raw)
			}
		})
	}
}

func TestGetDashboardStatsAggregatesTheCatalogue(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	customer := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	seedUser(t, db, "Bella Buyer", "bella@example.com", "user")

	// One product below the restock threshold, one comfortably above it.
	_, dress := seedCatalogue(t, db, "Silk Midi Dress", 5)
	seedCatalogue(t, db, "Wool Coat", 30)

	seedReview(t, db, customer, dress, 5)
	seedReview(t, db, customer, dress, 4)

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/dashboard/stats", "", adminToken(t, admin.ID))
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	stats := objectField(t, decodeJSON(t, raw), "stats")
	for _, tc := range []struct {
		field string
		want  float64
	}{
		{"total_users", 3},
		{"total_products", 2},
		{"total_categories", 2},
		{"total_reviews", 2},
		{"low_stock_products", 1},
		{"average_rating", 4.5},
	} {
		if got := numberField(t, stats, tc.field); got != tc.want {
			t.Errorf("expected %s %v, got %v", tc.field, tc.want, got)
		}
	}
}

func TestGetRecentActivityMergesEverySource(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	customer := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	_, dress := seedCatalogue(t, db, "Silk Midi Dress", 12)
	seedReview(t, db, customer, dress, 5)

	// The default limit is 20; asking for more than 100 is capped.
	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/dashboard/activity?limit=1000", "", adminToken(t, admin.ID))
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	activities := sliceField(t, body, "activities")
	// 2 users + 1 product + 1 review.
	if len(activities) != 4 {
		t.Fatalf("expected 4 activities, got %d: %s", len(activities), raw)
	}
	if count := numberField(t, body, "count"); count != 4 {
		t.Errorf("expected the count to match, got %v", count)
	}

	byType := make(map[string]map[string]interface{}, len(activities))
	for _, entry := range activities {
		activity := entry.(map[string]interface{})
		byType[stringField(t, activity, "type")] = activity
	}

	for _, want := range []string{"user_registered", "product_created", "review_added"} {
		if _, ok := byType[want]; !ok {
			t.Errorf("expected a %s activity, got %s", want, raw)
		}
	}

	if desc := stringField(t, byType["user_registered"], "description"); !strings.Contains(desc, "New user registered:") {
		t.Errorf("unexpected registration description %q", desc)
	}
	if desc := stringField(t, byType["product_created"], "description"); desc != "New product added: Silk Midi Dress" {
		t.Errorf("unexpected product description %q", desc)
	}
	if desc := stringField(t, byType["review_added"], "description"); desc != "Clara Client reviewed a product (5 stars)" {
		t.Errorf("unexpected review description %q", desc)
	}
	if id := numberField(t, byType["review_added"], "user_id"); uint(id) != customer.ID {
		t.Errorf("expected the reviewer id %d, got %v", customer.ID, id)
	}
}

func TestGetActivityLogsFiltersAndPaginates(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	token := adminToken(t, admin.ID)

	entries := []struct {
		adminID  uint
		action   string
		resource string
	}{
		{7, "created", "product"},
		{7, "updated", "product"},
		{8, "created", "user"},
	}
	for _, entry := range entries {
		if err := handlers.LogActivity(entry.adminID, "Ada Admin", entry.action, entry.resource, 1, "entry", "127.0.0.1"); err != nil {
			t.Fatalf("logging activity: %v", err)
		}
		// Keep created_at values distinct so the ordering is deterministic.
		time.Sleep(5 * time.Millisecond)
	}

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/dashboard/logs", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 3 {
		t.Errorf("expected 3 logs in total, got %v", total)
	}
	logs := sliceField(t, body, "logs")
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d: %s", len(logs), raw)
	}

	// Newest first: the last entry inserted is the "created user" one.
	newest := logs[0].(map[string]interface{})
	if action := stringField(t, newest, "action"); action != "created" {
		t.Errorf("expected the newest log first, got action %q", action)
	}
	if resource := stringField(t, newest, "resource"); resource != "user" {
		t.Errorf("expected the newest log to be the user one, got %q", resource)
	}
	if name := stringField(t, newest, "admin_name"); name != "Ada Admin" {
		t.Errorf("expected the admin name to be stored, got %q", name)
	}
	if ip := stringField(t, newest, "ip_address"); ip != "127.0.0.1" {
		t.Errorf("expected the IP to be stored, got %q", ip)
	}

	for _, tc := range []struct {
		name  string
		query string
		want  float64
	}{
		{"by action", "?action=created", 2},
		{"by resource", "?resource=user", 1},
		{"by admin", "?admin_id=7", 2},
		{"unknown admin", "?admin_id=99", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, "/api/v1/admin/dashboard/logs"+tc.query, "", token)
			if status != fiber.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", status, raw)
			}
			body := decodeJSON(t, raw)
			if total := numberField(t, body, "total"); total != tc.want {
				t.Errorf("expected total %v, got %v", tc.want, total)
			}
			if got := float64(len(sliceField(t, body, "logs"))); got != tc.want {
				t.Errorf("expected %v logs, got %v", tc.want, got)
			}
		})
	}

	// The page size is capped at 100.
	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/dashboard/logs?limit=1000", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if limit := numberField(t, decodeJSON(t, raw), "limit"); limit != 100 {
		t.Errorf("expected the limit to be capped at 100, got %v", limit)
	}

	// Second page of two.
	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/dashboard/logs?page=2&limit=2", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body = decodeJSON(t, raw)
	if page := numberField(t, body, "page"); page != 2 {
		t.Errorf("expected page 2, got %v", page)
	}
	if logs := sliceField(t, body, "logs"); len(logs) != 1 {
		t.Errorf("expected 1 log on the second page, got %d", len(logs))
	}
}
