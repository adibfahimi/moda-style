package handlers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/adibfahimi/moda-style/services/admin-service/models"
	"github.com/gofiber/fiber/v2"
)

func TestGetCategoriesCountsProductsAndResolvesParents(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")

	parent := testCategory{Name: "Women", Slug: "women"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("creating parent category: %v", err)
	}

	child := testCategory{Name: "Dresses", Slug: "dresses", ParentID: &parent.ID}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("creating child category: %v", err)
	}

	// One live product in the child, one soft-deleted product in the parent.
	if err := db.Create(&testProduct{Name: "Silk Midi Dress", CategoryID: child.ID, Price: 89.5}).Error; err != nil {
		t.Fatalf("creating product: %v", err)
	}
	deleted := testProduct{Name: "Old Coat", CategoryID: parent.ID, Price: 120}
	if err := db.Create(&deleted).Error; err != nil {
		t.Fatalf("creating product: %v", err)
	}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatalf("soft deleting product: %v", err)
	}

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/categories", "", adminToken(t, admin.ID))
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 2 {
		t.Errorf("expected 2 categories, got %v", total)
	}

	categories := sliceField(t, body, "categories")
	if len(categories) != 2 {
		t.Fatalf("expected 2 category rows, got %d: %s", len(categories), raw)
	}

	// Ordered by name: Dresses before Women.
	byName := make(map[string]map[string]interface{}, len(categories))
	for _, entry := range categories {
		category := entry.(map[string]interface{})
		byName[stringField(t, category, "name")] = category
	}

	dresses, ok := byName["Dresses"]
	if !ok {
		t.Fatalf("expected the child category in the list, got %s", raw)
	}
	if slug := stringField(t, dresses, "slug"); slug != "dresses" {
		t.Errorf("expected the slug to be returned, got %q", slug)
	}
	if name := stringField(t, dresses, "parent_name"); name != "Women" {
		t.Errorf("expected the parent name from the self join, got %q", name)
	}
	if count := numberField(t, dresses, "product_count"); count != 1 {
		t.Errorf("expected 1 product in the child, got %v", count)
	}

	women, ok := byName["Women"]
	if !ok {
		t.Fatalf("expected the parent category in the list, got %s", raw)
	}
	if count := numberField(t, women, "product_count"); count != 0 {
		t.Errorf("expected soft-deleted products to be ignored, got %v", count)
	}
	if _, hasParent := women["parent_name"]; hasParent {
		t.Errorf("expected a root category to omit parent_name, got %v", women)
	}

	if first := stringField(t, categories[0].(map[string]interface{}), "name"); first != "Dresses" {
		t.Errorf("expected the list to be ordered by name, got %q first", first)
	}
}

func TestGetCategoriesWithNoRowsIsAnEmptyArray(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/categories", "", adminToken(t, 1))
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 0 {
		t.Errorf("expected no categories, got %v", total)
	}
	if categories := sliceField(t, body, "categories"); len(categories) != 0 {
		t.Errorf("expected an empty array, got %v", categories)
	}
}

func TestCreateCategoryGeneratesSlugAndRejectsDuplicates(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	token := adminToken(t, admin.ID)
	const path = "/api/v1/admin/categories"

	for _, tc := range []struct {
		name string
		body string
	}{
		{"empty body", `{}`},
		{"short name", `{"name":"W"}`},
		{"malformed json", `{"name":`},
		{"unknown parent", `{"name":"Winter Wear","parent_id":4242}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, path, tc.body, token)
			if status != fiber.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
			}
		})
	}

	status, raw := request(t, app, http.MethodPost, path, `{"name":"Winter Wear"}`, token)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}
	created := objectField(t, decodeJSON(t, raw), "category")
	if slug := stringField(t, created, "slug"); slug != "winter-wear" {
		t.Errorf("expected the slug to be generated from the name, got %q", slug)
	}
	rootID := uint(numberField(t, created, "id"))

	var log models.ActivityLog
	if err := db.Where("resource = ? AND action = ?", "category", "created").First(&log).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if log.ResourceID != rootID || !strings.Contains(log.Description, "Winter Wear") {
		t.Errorf("unexpected audit log entry %+v", log)
	}

	for _, tc := range []struct {
		name string
		body string
	}{
		{"same name", `{"name":"Winter Wear"}`},
		{"name colliding on slug", `{"name":"Winter Wear!"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, path, tc.body, token)
			if status != fiber.StatusConflict {
				t.Fatalf("expected 409 Conflict, got %d: %s", status, raw)
			}
		})
	}

	// The same name is allowed under a different parent.
	status, raw = request(t, app, http.MethodPost, path,
		fmt.Sprintf(`{"name":"Winter Wear","parent_id":%d}`, rootID), token)
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}
	nested := objectField(t, decodeJSON(t, raw), "category")
	if parentID := numberField(t, nested, "parent_id"); uint(parentID) != rootID {
		t.Errorf("expected the parent to be stored, got %v", parentID)
	}
}

func TestUpdateCategoryGuardsTheHierarchy(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	token := adminToken(t, admin.ID)

	root := testCategory{Name: "Women", Slug: "women"}
	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("creating root category: %v", err)
	}
	child := testCategory{Name: "Dresses", Slug: "dresses", ParentID: &root.ID}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("creating child category: %v", err)
	}

	childPath := fmt.Sprintf("/api/v1/admin/categories/%d", child.ID)
	rootPath := fmt.Sprintf("/api/v1/admin/categories/%d", root.ID)

	for _, tc := range []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{"malformed id", "/api/v1/admin/categories/abc", `{"name":"Skirts"}`, fiber.StatusBadRequest},
		{"unknown category", "/api/v1/admin/categories/4242", `{"name":"Skirts"}`, fiber.StatusNotFound},
		{"invalid body", childPath, `{}`, fiber.StatusBadRequest},
		{"own parent", childPath, fmt.Sprintf(`{"name":"Dresses","parent_id":%d}`, child.ID), fiber.StatusBadRequest},
		{"unknown parent", childPath, `{"name":"Dresses","parent_id":4242}`, fiber.StatusBadRequest},
		{"descendant as parent", rootPath, fmt.Sprintf(`{"name":"Women","parent_id":%d}`, child.ID), fiber.StatusBadRequest},
		{"name taken", childPath, `{"name":"Women"}`, fiber.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPut, tc.path, tc.body, token)
			if status != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, status, raw)
			}
		})
	}

	// A successful update promotes the child to a root category.
	status, raw := request(t, app, http.MethodPut, childPath,
		`{"name":"Summer Dresses","slug":"summer-dresses"}`, token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if name := stringField(t, objectField(t, decodeJSON(t, raw), "category"), "name"); name != "Summer Dresses" {
		t.Errorf("expected the new name, got %q", name)
	}

	var stored testCategory
	if err := db.First(&stored, child.ID).Error; err != nil {
		t.Fatalf("loading category: %v", err)
	}
	if stored.Name != "Summer Dresses" || stored.Slug != "summer-dresses" {
		t.Errorf("expected the category to be updated, got %+v", stored)
	}
	if stored.ParentID != nil {
		t.Errorf("expected the parent to be cleared, got %v", *stored.ParentID)
	}

	var log models.ActivityLog
	if err := db.Where("resource = ? AND action = ?", "category", "updated").First(&log).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if log.ResourceID != child.ID {
		t.Errorf("expected the log to point at the category, got %d", log.ResourceID)
	}
}

func TestDeleteCategoryRefusesProductsAndChildren(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	token := adminToken(t, admin.ID)

	parent := testCategory{Name: "Women", Slug: "women"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("creating parent category: %v", err)
	}
	child := testCategory{Name: "Dresses", Slug: "dresses", ParentID: &parent.ID}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("creating child category: %v", err)
	}
	if err := db.Create(&testProduct{Name: "Silk Midi Dress", CategoryID: child.ID, Price: 89.5}).Error; err != nil {
		t.Fatalf("creating product: %v", err)
	}

	for _, tc := range []struct {
		name   string
		path   string
		status int
	}{
		{"malformed id", "/api/v1/admin/categories/abc", fiber.StatusBadRequest},
		{"unknown category", "/api/v1/admin/categories/4242", fiber.StatusNotFound},
		{"category with products", fmt.Sprintf("/api/v1/admin/categories/%d", child.ID), fiber.StatusBadRequest},
		{"category with children", fmt.Sprintf("/api/v1/admin/categories/%d", parent.ID), fiber.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodDelete, tc.path, "", token)
			if status != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, status, raw)
			}
		})
	}

	// Removing the product first unblocks the child category.
	if err := db.Where("category_id = ?", child.ID).Delete(&testProduct{}).Error; err != nil {
		t.Fatalf("soft deleting product: %v", err)
	}

	status, raw := request(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/categories/%d", child.ID), "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "Category deleted successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	var remaining int64
	if err := db.Table("categories").Where("id = ? AND deleted_at IS NULL", child.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("counting categories: %v", err)
	}
	if remaining != 0 {
		t.Errorf("expected the category to be soft deleted, got %d rows", remaining)
	}

	// With the child gone the root category can be removed as well, and the
	// listing is empty afterwards.
	status, raw = request(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/categories/%d", parent.ID), "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/categories", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if categories := sliceField(t, decodeJSON(t, raw), "categories"); len(categories) != 0 {
		t.Errorf("expected no categories left, got %v", categories)
	}

	var log models.ActivityLog
	if err := db.Where("resource = ? AND action = ?", "category", "deleted").
		Order("id ASC").First(&log).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if !strings.Contains(log.Description, "Dresses") {
		t.Errorf("expected the deleted category name in the log, got %q", log.Description)
	}
}
