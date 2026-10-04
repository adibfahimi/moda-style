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
	"github.com/adibfahimi/moda-style/services/product-service/database"
	"github.com/adibfahimi/moda-style/services/product-service/handlers"
	"github.com/adibfahimi/moda-style/services/product-service/models"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testJWTSecret is the signing key every test in this package uses. TestMain
// installs it before any test runs because common resolves JWT_SECRET lazily.
const testJWTSecret = "product-service-test-secret"

// dbCounter hands every test its own in-memory database.
var dbCounter atomic.Uint64

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

// newTestDB returns an isolated in-memory SQLite database with the catalogue
// schema migrated and points database.DB at it.
//
// SQLite keeps the handler tests self-contained. The SQL used by this service
// (joins, LOWER/LIKE, AVG, soft deletes) is dialect neutral, so it runs
// unchanged on PostgreSQL in production.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:product_test_%d?mode=memory&cache=shared", dbCounter.Add(1))
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

	if err := db.AutoMigrate(&models.Category{}, &models.Product{}, &models.Size{}, &models.Review{}); err != nil {
		t.Fatalf("migrating catalogue schema: %v", err)
	}

	database.DB = db
	return db
}

// newTestApp mounts the same routes as services/product-service/main.go.
func newTestApp() *fiber.App {
	app := fiber.New()

	api := app.Group("/api/v1")
	api.Get("/products", handlers.ListProducts)
	api.Get("/products/:id", handlers.GetProduct)
	api.Get("/categories", handlers.ListCategories)
	api.Get("/products/:id/reviews", handlers.GetProductReviews)
	api.Post("/products/:id/reviews", common.RequireAuth, handlers.CreateReview)

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
func tokenFor(t *testing.T, userID uint, name string) string {
	t.Helper()

	token, err := common.GenerateToken(userID, fmt.Sprintf("user%d@example.com", userID), name, "user")
	if err != nil {
		t.Fatalf("generating token: %v", err)
	}
	return token
}

// createCategory inserts a category, optionally attached to a parent.
func createCategory(t *testing.T, db *gorm.DB, name, slug string, parentID *uint) models.Category {
	t.Helper()

	category := models.Category{Name: name, Slug: slug, ParentID: parentID}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("creating category %q: %v", name, err)
	}
	return category
}

// createProduct inserts a product in the given category.
func createProduct(t *testing.T, db *gorm.DB, name, description string, price float64, categoryID uint) models.Product {
	t.Helper()

	product := models.Product{Name: name, Description: description, Price: price, CategoryID: categoryID}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("creating product %q: %v", name, err)
	}
	return product
}

// addSize inserts one size/colour variant of a product with the given stock.
func addSize(t *testing.T, db *gorm.DB, productID uint, size, color string, stock int) {
	t.Helper()

	variant := models.Size{ProductID: productID, Size: size, Color: color, Stock: stock}
	if err := db.Create(&variant).Error; err != nil {
		t.Fatalf("creating size %s/%s of product %d: %v", size, color, productID, err)
	}
}

// seededCatalog is the fixture shared by the catalogue tests.
type seededCatalog struct {
	women   models.Category
	dresses models.Category
	men     models.Category
	dress   models.Product
	jacket  models.Product
	coat    models.Product
}

// seedCatalog creates a two-level category tree with three products:
//
//	Women (root)
//	  └── Dresses  → Silk Midi Dress   (M/Black 3, L/Black 0)
//	Men (root)     → Leather Jacket    (M/Brown 5)
//	Women          → Wool Coat         (XL/Grey 2)
//
// The nested dress and the out-of-stock L variant make the hierarchy, stock and
// filter assertions meaningful.
func seedCatalog(t *testing.T, db *gorm.DB) seededCatalog {
	t.Helper()

	women := createCategory(t, db, "Women", "women", nil)
	dresses := createCategory(t, db, "Dresses", "dresses", &women.ID)
	men := createCategory(t, db, "Men", "men", nil)

	dress := createProduct(t, db, "Silk Midi Dress", "Flowing silk dress for evening wear", 89.5, dresses.ID)
	addSize(t, db, dress.ID, "M", "Black", 3)
	addSize(t, db, dress.ID, "L", "Black", 0)

	jacket := createProduct(t, db, "Leather Jacket", "Cropped leather jacket", 249, men.ID)
	addSize(t, db, jacket.ID, "M", "Brown", 5)

	coat := createProduct(t, db, "Wool Coat", "Warm wool coat for winter", 180, women.ID)
	addSize(t, db, coat.ID, "XL", "Grey", 2)

	return seededCatalog{women: women, dresses: dresses, men: men, dress: dress, jacket: jacket, coat: coat}
}

func TestListCategoriesReturnsParentsBeforeChildren(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalog(t, db)

	status, raw := request(t, app, http.MethodGet, "/api/v1/categories", "", "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	categories := sliceField(t, decodeJSON(t, raw), "categories")
	if len(categories) != 3 {
		t.Fatalf("expected 3 categories, got %d: %s", len(categories), raw)
	}

	first := categories[0].(map[string]interface{})
	if _, hasParent := first["parent_id"]; hasParent {
		t.Errorf("expected root categories first, got %v", first)
	}

	// The child category must expose the preloaded parent.
	var child map[string]interface{}
	for _, entry := range categories {
		category := entry.(map[string]interface{})
		if category["name"] == "Dresses" {
			child = category
		}
	}
	if child == nil {
		t.Fatalf("expected the Dresses category in %s", raw)
	}
	if parent := objectField(t, child, "parent"); numberField(t, parent, "id") != float64(seed.women.ID) {
		t.Errorf("expected Dresses to be attached to Women, got %v", parent)
	}
}

func TestListProductsComputesStockAndPreloadsRelations(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seedCatalog(t, db)

	status, raw := request(t, app, http.MethodGet, "/api/v1/products", "", "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 3 {
		t.Errorf("expected total 3, got %v", total)
	}
	if page := numberField(t, body, "page"); page != 1 {
		t.Errorf("expected default page 1, got %v", page)
	}
	if limit := numberField(t, body, "limit"); limit != 20 {
		t.Errorf("expected default limit 20, got %v", limit)
	}

	products := sliceField(t, body, "products")
	if len(products) != 3 {
		t.Fatalf("expected 3 products, got %d: %s", len(products), raw)
	}

	for _, entry := range products {
		product := entry.(map[string]interface{})
		if product["name"] != "Silk Midi Dress" {
			continue
		}
		// Stock is derived from the sizes: 3 (M) + 0 (L).
		if stock := numberField(t, product, "stock"); stock != 3 {
			t.Errorf("expected computed stock 3, got %v", stock)
		}
		if category := objectField(t, product, "category"); category["name"] != "Dresses" {
			t.Errorf("expected the Dresses category to be preloaded, got %v", category)
		}
		if sizes := sliceField(t, product, "sizes"); len(sizes) != 2 {
			t.Errorf("expected 2 size variants, got %d", len(sizes))
		}
	}
}

func TestListProductsFilters(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalog(t, db)

	tests := []struct {
		name      string
		query     string
		wantNames []string
	}{
		{
			// A numeric category ID must include products filed in subcategories.
			name:      "category id includes descendants",
			query:     fmt.Sprintf("?category=%d", seed.women.ID),
			wantNames: []string{"Silk Midi Dress", "Wool Coat"},
		},
		{
			name:      "category name matches direct children only",
			query:     "?category=Dresses",
			wantNames: []string{"Silk Midi Dress"},
		},
		{
			name:      "size filter requires stock",
			query:     "?size=M",
			wantNames: []string{"Silk Midi Dress", "Leather Jacket"},
		},
		{
			name:      "size filter ignores out of stock variants",
			query:     "?size=L",
			wantNames: nil,
		},
		{
			name:      "colour filter requires stock",
			query:     "?color=Brown",
			wantNames: []string{"Leather Jacket"},
		},
		{
			name:      "price range is inclusive",
			query:     "?min_price=100&max_price=200",
			wantNames: []string{"Wool Coat"},
		},
		{
			name:      "search is case insensitive over the name",
			query:     "?search=SILK",
			wantNames: []string{"Silk Midi Dress"},
		},
		{
			name:      "search also matches the description",
			query:     "?search=winter",
			wantNames: []string{"Wool Coat"},
		},
		{
			name:      "unknown category name yields no products",
			query:     "?category=Underwear",
			wantNames: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, "/api/v1/products"+tc.query, "", "")
			if status != fiber.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", status, raw)
			}

			products := sliceField(t, decodeJSON(t, raw), "products")
			if len(products) != len(tc.wantNames) {
				t.Fatalf("expected %d products %v, got %d: %s", len(tc.wantNames), tc.wantNames, len(products), raw)
			}

			got := make(map[string]bool, len(products))
			for _, entry := range products {
				got[fmt.Sprint(entry.(map[string]interface{})["name"])] = true
			}
			for _, name := range tc.wantNames {
				if !got[name] {
					t.Errorf("expected product %q in %s", name, raw)
				}
			}
		})
	}
}

func TestListProductsPaginatesAndClampsPagingInput(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seedCatalog(t, db)

	status, raw := request(t, app, http.MethodGet, "/api/v1/products?page=2&limit=2", "", "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body := decodeJSON(t, raw)
	if products := sliceField(t, body, "products"); len(products) != 1 {
		t.Errorf("expected 1 product on the second page, got %d: %s", len(products), raw)
	}
	if total := numberField(t, body, "total"); total != 3 {
		t.Errorf("total must ignore pagination, got %v", total)
	}

	// Out-of-range paging values fall back to the defaults instead of erroring.
	status, raw = request(t, app, http.MethodGet, "/api/v1/products?page=0&limit=1000", "", "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body = decodeJSON(t, raw)
	if page := numberField(t, body, "page"); page != 1 {
		t.Errorf("expected page to be clamped to 1, got %v", page)
	}
	if limit := numberField(t, body, "limit"); limit != 20 {
		t.Errorf("expected limit to be clamped to 20, got %v", limit)
	}
}

func TestGetProductReturnsSizesAndComputedStock(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalog(t, db)

	status, raw := request(t, app, http.MethodGet, fmt.Sprintf("/api/v1/products/%d", seed.dress.ID), "", "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	product := objectField(t, decodeJSON(t, raw), "product")
	if name := stringField(t, product, "name"); name != "Silk Midi Dress" {
		t.Errorf("unexpected product %q", name)
	}
	if stock := numberField(t, product, "stock"); stock != 3 {
		t.Errorf("expected computed stock 3, got %v", stock)
	}
	if objectField(t, product, "category")["name"] != "Dresses" {
		t.Errorf("expected the category to be preloaded, got %v", product["category"])
	}
}

func TestGetProductReturnsNotFoundForUnknownID(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	status, raw := request(t, app, http.MethodGet, "/api/v1/products/4242", "", "")
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Product not found" {
		t.Errorf("unexpected error message %q", msg)
	}
}
