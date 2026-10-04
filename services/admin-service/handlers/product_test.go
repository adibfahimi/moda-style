package handlers_test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adibfahimi/moda-style/services/admin-service/models"
	"github.com/gofiber/fiber/v2"
)

func TestCreateProductValidatesCategoryAndLogs(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	category, _ := seedCatalogue(t, db, "Silk Midi Dress", 12)
	token := adminToken(t, admin.ID)

	for _, tc := range []struct {
		name string
		body string
	}{
		{"empty body", `{}`},
		{"short name", `{"name":"X","price":10,"category_id":1}`},
		{"zero price", fmt.Sprintf(`{"name":"Wool Coat","price":0,"category_id":%d}`, category.ID)},
		{"missing category", `{"name":"Wool Coat","price":10}`},
		{"unknown category", `{"name":"Wool Coat","price":10,"category_id":999}`},
		{"malformed json", `{"name":`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, "/api/v1/admin/products", tc.body, token)
			if status != fiber.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
			}
			if msg := stringField(t, decodeJSON(t, raw), "error"); msg == "" {
				t.Errorf("expected an error message, got %s", raw)
			}
		})
	}

	status, raw := request(t, app, http.MethodPost, "/api/v1/admin/products",
		fmt.Sprintf(`{"name":"Wool Coat","description":"Warm winter coat","price":149.9,"category_id":%d,"image_url":"/uploads/products/coat.png"}`, category.ID),
		token)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Product created successfully" {
		t.Errorf("unexpected message %q", msg)
	}
	product := objectField(t, body, "product")
	if price := numberField(t, product, "price"); price != 149.9 {
		t.Errorf("expected the price to be echoed, got %v", price)
	}

	var created testProduct
	if err := db.Where("name = ?", "Wool Coat").First(&created).Error; err != nil {
		t.Fatalf("expected the product to be stored: %v", err)
	}
	if created.CategoryID != category.ID {
		t.Errorf("expected category %d, got %d", category.ID, created.CategoryID)
	}

	var log models.ActivityLog
	if err := db.Where("action = ? AND resource = ?", "created", "product").First(&log).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if !strings.Contains(log.Description, "Wool Coat") || log.ResourceID != created.ID {
		t.Errorf("unexpected audit log %+v", log)
	}
}

func TestUpdateProductExistenceAndCategory(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	category, product := seedCatalogue(t, db, "Silk Midi Dress", 12)
	token := adminToken(t, admin.ID)
	path := fmt.Sprintf("/api/v1/admin/products/%d", product.ID)

	status, raw := request(t, app, http.MethodPut, "/api/v1/admin/products/not-a-number",
		fmt.Sprintf(`{"name":"Wool Coat","price":10,"category_id":%d}`, category.ID), token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodPut, "/api/v1/admin/products/4242",
		fmt.Sprintf(`{"name":"Wool Coat","price":10,"category_id":%d}`, category.ID), token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodPut, path,
		`{"name":"Wool Coat","price":10,"category_id":999}`, token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for an unknown category, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodPut, path,
		fmt.Sprintf(`{"name":"Silk Maxi Dress","description":"Longer cut","price":99.5,"category_id":%d}`, category.ID),
		token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "Product updated successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	var updated testProduct
	if err := db.First(&updated, product.ID).Error; err != nil {
		t.Fatalf("loading product: %v", err)
	}
	if updated.Name != "Silk Maxi Dress" {
		t.Errorf("expected the name to be updated, got %q", updated.Name)
	}
	if updated.Price != 99.5 {
		t.Errorf("expected the price to be updated, got %v", updated.Price)
	}

	var log models.ActivityLog
	if err := db.Where("action = ? AND resource = ?", "updated", "product").First(&log).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if log.ResourceID != product.ID {
		t.Errorf("expected the updated product %d in the audit log, got %d", product.ID, log.ResourceID)
	}
}

func TestDeleteProductSoftDeletesAndRejectsUnknownIDs(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	_, product := seedCatalogue(t, db, "Silk Midi Dress", 12)
	token := adminToken(t, admin.ID)

	status, raw := request(t, app, http.MethodDelete, "/api/v1/admin/products/not-a-number", "", token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}

	// Regression: an unknown id used to answer 200 because Scan does not report
	// a missing row as an error.
	status, raw = request(t, app, http.MethodDelete, "/api/v1/admin/products/4242", "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/products/%d", product.ID), "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "Product deleted successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	var softDeleted int64
	if err := db.Table("products").
		Where("id = ? AND deleted_at IS NOT NULL", product.ID).
		Count(&softDeleted).Error; err != nil {
		t.Fatalf("counting deleted products: %v", err)
	}
	if softDeleted != 1 {
		t.Errorf("expected the product to be soft deleted, got %d rows", softDeleted)
	}

	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/products", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if total := numberField(t, decodeJSON(t, raw), "total"); total != 0 {
		t.Errorf("expected the deleted product to disappear, got %v", total)
	}
}

func TestGetProductStatsAggregatesSizesReviewsAndWishlist(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	customer := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	_, dress := seedCatalogue(t, db, "Silk Midi Dress", 5)
	seedCatalogue(t, db, "Wool Coat", 30)

	// A second size variant for the dress: stock is summed per product.
	if err := db.Create(&testSize{ProductID: dress.ID, Size: "L", Color: "Black", Stock: 7}).Error; err != nil {
		t.Fatalf("creating size: %v", err)
	}
	seedReview(t, db, customer, dress, 5)
	seedReview(t, db, customer, dress, 3)
	if err := db.Create(&testWishlistItem{UserID: customer.ID, ProductID: dress.ID}).Error; err != nil {
		t.Fatalf("creating wishlist item: %v", err)
	}

	token := adminToken(t, admin.ID)
	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/products", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 2 {
		t.Errorf("expected 2 products, got %v", total)
	}

	products := sliceField(t, body, "products")
	if len(products) != 2 {
		t.Fatalf("expected 2 product rows, got %d: %s", len(products), raw)
	}

	byName := make(map[string]map[string]interface{}, len(products))
	for _, entry := range products {
		product := entry.(map[string]interface{})
		byName[stringField(t, product, "name")] = product
	}

	stats, ok := byName["Silk Midi Dress"]
	if !ok {
		t.Fatalf("expected the dress in the list, got %s", raw)
	}
	if stock := numberField(t, stats, "stock"); stock != 12 {
		t.Errorf("expected the sizes to be summed to 12, got %v", stock)
	}
	if count := numberField(t, stats, "review_count"); count != 2 {
		t.Errorf("expected 2 reviews, got %v", count)
	}
	if avg := numberField(t, stats, "average_rating"); avg != 4 {
		t.Errorf("expected an average rating of 4, got %v", avg)
	}
	if count := numberField(t, stats, "wishlist_count"); count != 1 {
		t.Errorf("expected 1 wishlist entry, got %v", count)
	}
	if name := stringField(t, stats, "category_name"); name != "Dresses" {
		t.Errorf("expected the category name from the join, got %q", name)
	}

	coat, ok := byName["Wool Coat"]
	if !ok {
		t.Fatalf("expected the coat in the list, got %s", raw)
	}
	if count := numberField(t, coat, "review_count"); count != 0 {
		t.Errorf("expected no reviews for the coat, got %v", count)
	}

	// Pagination: the second page holds a single product.
	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/products?page=2&limit=1", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	page2 := decodeJSON(t, raw)
	if page := numberField(t, page2, "page"); page != 2 {
		t.Errorf("expected page 2, got %v", page)
	}
	if products := sliceField(t, page2, "products"); len(products) != 1 {
		t.Errorf("expected 1 product on the second page, got %d", len(products))
	}

	// The page size is capped at 100, and an empty page is an array, not null.
	for _, tc := range []struct {
		name  string
		query string
		check func(t *testing.T, body map[string]interface{})
	}{
		{
			name:  "limit is capped",
			query: "?limit=500",
			check: func(t *testing.T, body map[string]interface{}) {
				if limit := numberField(t, body, "limit"); limit != 100 {
					t.Errorf("expected the limit to be capped at 100, got %v", limit)
				}
			},
		},
		{
			name:  "empty page",
			query: "?page=99",
			check: func(t *testing.T, body map[string]interface{}) {
				if products := sliceField(t, body, "products"); len(products) != 0 {
					t.Errorf("expected an empty array, got %v", products)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, "/api/v1/admin/products"+tc.query, "", token)
			if status != fiber.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", status, raw)
			}
			tc.check(t, decodeJSON(t, raw))
		})
	}
}

func TestProductSizeLifecycle(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	_, product := seedCatalogue(t, db, "Silk Midi Dress", 5)
	token := adminToken(t, admin.ID)

	sizesPath := fmt.Sprintf("/api/v1/admin/products/%d/sizes", product.ID)

	// Listing sizes of an unknown product is a 404, of a known product an array.
	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/products/4242/sizes", "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
	status, raw = request(t, app, http.MethodGet, sizesPath, "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 1 {
		t.Errorf("expected the seeded size, got %v", total)
	}

	// Validation: empty body, a negative stock and an unknown product.
	for _, tc := range []struct {
		name   string
		body   string
		status int
	}{
		{"empty body", `{}`, fiber.StatusBadRequest},
		{"negative stock", `{"size":"L","color":"Black","stock":-1}`, fiber.StatusBadRequest},
		{"missing color", `{"size":"L","stock":3}`, fiber.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, sizesPath, tc.body, token)
			if status != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, status, raw)
			}
		})
	}

	status, raw = request(t, app, http.MethodPost, "/api/v1/admin/products/4242/sizes",
		`{"size":"L","color":"Black","stock":3}`, token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
	status, raw = request(t, app, http.MethodPost, "/api/v1/admin/products/not-a-number/sizes",
		`{"size":"L","color":"Black","stock":3}`, token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodPost, sizesPath,
		`{"size":"L","color":"Black","stock":3}`, token)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}
	created := objectField(t, decodeJSON(t, raw), "size")
	sizeID := numberField(t, created, "id")
	if productID := numberField(t, created, "product_id"); uint(productID) != product.ID {
		t.Errorf("expected the size to belong to product %d, got %v", product.ID, productID)
	}

	var sizeLog models.ActivityLog
	if err := db.Where("resource = ? AND action = ?", "size", "created").First(&sizeLog).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if sizeLog.ResourceID != uint(sizeID) {
		t.Errorf("expected the size %v in the audit log, got %d", sizeID, sizeLog.ResourceID)
	}

	// Updating the new variant.
	putPath := fmt.Sprintf("%s/%d", sizesPath, uint(sizeID))
	status, raw = request(t, app, http.MethodPut, fmt.Sprintf("%s/not-a-number", sizesPath),
		`{"size":"L","color":"Black","stock":3}`, token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	status, raw = request(t, app, http.MethodPut, fmt.Sprintf("%s/4242", sizesPath),
		`{"size":"L","color":"Black","stock":3}`, token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
	status, raw = request(t, app, http.MethodPut, putPath,
		`{"size":"XL","color":"Ivory","stock":9}`, token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	var updated testSize
	if err := db.First(&updated, uint(sizeID)).Error; err != nil {
		t.Fatalf("loading size: %v", err)
	}
	if updated.Size != "XL" || updated.Color != "Ivory" || updated.Stock != 9 {
		t.Errorf("expected the size to be updated, got %+v", updated)
	}

	// Deleting the variant soft deletes it.
	status, raw = request(t, app, http.MethodDelete, fmt.Sprintf("%s/4242", sizesPath), "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
	status, raw = request(t, app, http.MethodDelete, putPath, "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	var softDeleted int64
	if err := db.Table("sizes").
		Where("id = ? AND deleted_at IS NOT NULL", uint(sizeID)).
		Count(&softDeleted).Error; err != nil {
		t.Fatalf("counting deleted sizes: %v", err)
	}
	if softDeleted != 1 {
		t.Errorf("expected the size to be soft deleted, got %d rows", softDeleted)
	}
}

// pngHeader is the magic prefix http.DetectContentType recognises as a PNG.
var pngHeader = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)

// multipartRequest posts a single file field, mirroring request for the upload
// endpoints that expect multipart/form-data.
func multipartRequest(t *testing.T, app *fiber.App, path, field, filename string, content []byte, token string) (int, []byte) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("creating multipart part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("writing multipart part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

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

func TestUploadProductImageStoresFilesAndValidatesThem(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	// The handler writes into ./uploads/products relative to the working
	// directory, so the test runs inside a throwaway directory.
	t.Chdir(t.TempDir())

	token := adminToken(t, 1)
	const path = "/api/v1/admin/products/upload-image"

	status, raw := multipartRequest(t, app, path, "image", "coat.png", pngHeader, token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Image uploaded successfully" {
		t.Errorf("unexpected message %q", msg)
	}
	imagePath := stringField(t, body, "image_path")
	if !strings.HasPrefix(imagePath, "/api/v1/admin/uploads/products/product-") {
		t.Errorf("unexpected image path %q", imagePath)
	}

	fileName := filepath.Base(imagePath)
	if _, err := os.Stat(filepath.Join("uploads", "products", fileName)); err != nil {
		t.Fatalf("expected the image on disk: %v", err)
	}

	// A file whose extension is not allowed is rejected.
	status, raw = multipartRequest(t, app, path, "image", "coat.gif", pngHeader, token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Only JPG, PNG, and WEBP files are allowed" {
		t.Errorf("unexpected error message %q", msg)
	}

	// A .png file that is not actually an image is rejected too.
	status, raw = multipartRequest(t, app, path, "image", "fake.png", []byte("not an image at all"), token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); !strings.Contains(msg, "Invalid image file type") {
		t.Errorf("unexpected error message %q", msg)
	}

	// The listing ignores files that are not images but accepts any casing of
	// the known extensions.
	if err := os.WriteFile(filepath.Join("uploads", "products", "COAT.JPG"), pngHeader, 0o600); err != nil {
		t.Fatalf("writing uppercase file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join("uploads", "products", "drafts"), 0o750); err != nil {
		t.Fatalf("creating subdirectory: %v", err)
	}
	if err := os.WriteFile(filepath.Join("uploads", "products", "notes.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatalf("writing stray file: %v", err)
	}

	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/products/images", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body = decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 2 {
		t.Errorf("expected 2 images, got %v", total)
	}

	images := sliceField(t, body, "images")
	byName := make(map[string]map[string]interface{}, len(images))
	for _, entry := range images {
		image := entry.(map[string]interface{})
		byName[stringField(t, image, "name")] = image
	}

	uploaded, ok := byName[fileName]
	if !ok {
		t.Fatalf("expected %q in the listing, got %v", fileName, byName)
	}
	if url := stringField(t, uploaded, "url"); !strings.HasSuffix(url, "/api/v1/admin/uploads/products/"+fileName) {
		t.Errorf("unexpected url %q", url)
	}
	if size := numberField(t, uploaded, "size"); size != float64(len(pngHeader)) {
		t.Errorf("expected the file size %d, got %v", len(pngHeader), size)
	}
	if updatedAt := stringField(t, uploaded, "updated_at"); updatedAt == "" {
		t.Error("expected an updated_at timestamp")
	}
	if _, ok := byName["COAT.JPG"]; !ok {
		t.Errorf("expected the uppercase extension to be listed, got %v", byName)
	}
	if _, ok := byName["notes.txt"]; ok {
		t.Errorf("expected the non-image to be ignored, got %v", byName)
	}
}
