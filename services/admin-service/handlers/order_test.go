package handlers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/adibfahimi/moda-style/services/admin-service/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// seedOrderAt inserts an order and pins its created_at so the ordering and the
// time window analytics are deterministic.
func seedOrderAt(t *testing.T, db *gorm.DB, user testUser, number, status, paymentStatus string, total float64, createdAt time.Time) testOrder {
	t.Helper()

	order, _ := seedOrder(t, db, user, number, status, paymentStatus, total)
	if err := db.Model(&testOrder{}).Where("id = ?", order.ID).
		Update("created_at", createdAt).Error; err != nil {
		t.Fatalf("pinning created_at: %v", err)
	}
	order.CreatedAt = createdAt
	return order
}

func TestGetOrdersFiltersSearchesAndPaginates(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	bob := seedUser(t, db, "Bob Buyer", "bob@example.com", "user")
	token := adminToken(t, admin.ID)

	now := time.Now()
	seedOrderAt(t, db, clara, "ORD-1001", "pending", "paid", 100, now.Add(-3*time.Hour))
	seedOrderAt(t, db, clara, "ORD-1002", "shipped", "paid", 200, now.Add(-2*time.Hour))
	seedOrderAt(t, db, bob, "ORD-1003", "cancelled", "pending", 50, now.Add(-1*time.Hour))

	// Soft-deleted orders must not show up.
	removed, _ := seedOrder(t, db, clara, "ORD-1004", "pending", "pending", 10)
	if err := db.Delete(&removed).Error; err != nil {
		t.Fatalf("soft deleting order: %v", err)
	}

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/orders?limit=2", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 3 {
		t.Errorf("expected 3 live orders, got %v", total)
	}
	if limit := numberField(t, body, "limit"); limit != 2 {
		t.Errorf("expected the limit to be echoed, got %v", limit)
	}

	orders := sliceField(t, body, "orders")
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders on the first page, got %d: %s", len(orders), raw)
	}

	// Newest first, and the user's name/email come from the join.
	first := orders[0].(map[string]interface{})
	if number := stringField(t, first, "order_number"); number != "ORD-1003" {
		t.Errorf("expected the most recent order first, got %q", number)
	}
	if name := stringField(t, first, "user_name"); name != "Bob Buyer" {
		t.Errorf("expected the joined user name, got %q", name)
	}
	if email := stringField(t, first, "user_email"); email != "bob@example.com" {
		t.Errorf("expected the joined user email, got %q", email)
	}
	if count := numberField(t, first, "item_count"); count != 1 {
		t.Errorf("expected the line item to be counted, got %v", count)
	}
	if amount := numberField(t, first, "total_amount"); amount != 50 {
		t.Errorf("expected the order total, got %v", amount)
	}

	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/orders?limit=2&page=2", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	page2 := decodeJSON(t, raw)
	page2Orders := sliceField(t, page2, "orders")
	if len(page2Orders) != 1 {
		t.Fatalf("expected 1 order on the second page, got %d: %s", len(page2Orders), raw)
	}
	if number := stringField(t, page2Orders[0].(map[string]interface{}), "order_number"); number != "ORD-1001" {
		t.Errorf("expected the oldest order last, got %q", number)
	}

	for _, tc := range []struct {
		name      string
		query     string
		wantTotal float64
		wantEmpty bool
	}{
		{"status filter", "?status=shipped", 1, false},
		{"payment status filter", "?payment_status=paid", 2, false},
		{"unknown filter", "?status=refunded", 0, true},
		{"search by order number", "?search=ORD-1002", 1, false},
		{"search by customer name", "?search=buyer", 1, false},
		{"search without match", "?search=nobody", 0, true},
		{"out of range limit falls back", "?limit=500", 3, false},
		{"zero page falls back", "?page=0", 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, "/api/v1/admin/orders"+tc.query, "", token)
			if status != fiber.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", status, raw)
			}

			body := decodeJSON(t, raw)
			if total := numberField(t, body, "total"); total != tc.wantTotal {
				t.Errorf("expected a total of %v, got %v (%s)", tc.wantTotal, total, raw)
			}
			if page := numberField(t, body, "page"); page < 1 {
				t.Errorf("expected a normalised page number, got %v", page)
			}
			if limit := numberField(t, body, "limit"); limit != 20 {
				t.Errorf("expected the default limit of 20, got %v", limit)
			}

			orders := sliceField(t, body, "orders")
			if tc.wantEmpty && len(orders) != 0 {
				t.Errorf("expected an empty array, got %v", orders)
			}
			if !tc.wantEmpty && len(orders) == 0 {
				t.Errorf("expected orders, got %s", raw)
			}
		})
	}

	// An out of range limit is clamped back to the default, not to 100.
	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/orders?limit=101", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if limit := numberField(t, decodeJSON(t, raw), "limit"); limit != 20 {
		t.Errorf("expected an out of range limit to fall back to 20, got %v", limit)
	}
}

func TestGetOrderDetailsReturnsItemsAndGuardsUnknownIDs(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	token := adminToken(t, admin.ID)

	order, item := seedOrder(t, db, clara, "ORD-2001", "processing", "paid", 89.5)
	if err := db.Create(&testOrderItem{
		OrderID: order.ID, ProductID: 2, ProductName: "Wool Coat", Quantity: 2, Price: 120, Subtotal: 240,
	}).Error; err != nil {
		t.Fatalf("creating second order item: %v", err)
	}

	for _, tc := range []struct {
		name   string
		path   string
		status int
	}{
		{"malformed id", "/api/v1/admin/orders/abc", fiber.StatusBadRequest},
		{"unknown order", "/api/v1/admin/orders/4242", fiber.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, tc.path, "", token)
			if status != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, status, raw)
			}
		})
	}

	status, raw := request(t, app, http.MethodGet, fmt.Sprintf("/api/v1/admin/orders/%d", order.ID), "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	details := objectField(t, body, "order")
	if number := stringField(t, details, "order_number"); number != "ORD-2001" {
		t.Errorf("expected the order number, got %q", number)
	}
	if name := stringField(t, details, "user_name"); name != "Clara Client" {
		t.Errorf("expected the joined customer name, got %q", name)
	}
	if email := stringField(t, details, "user_email"); email != "clara@example.com" {
		t.Errorf("expected the joined customer email, got %q", email)
	}
	if count := numberField(t, details, "item_count"); count != 2 {
		t.Errorf("expected both line items to be counted, got %v", count)
	}

	items := sliceField(t, body, "items")
	if len(items) != 2 {
		t.Fatalf("expected 2 preloaded items, got %d: %s", len(items), raw)
	}
	names := []string{
		stringField(t, items[0].(map[string]interface{}), "product_name"),
		stringField(t, items[1].(map[string]interface{}), "product_name"),
	}
	if !strings.Contains(strings.Join(names, ","), "Wool Coat") {
		t.Errorf("expected the preloaded items, got %v", names)
	}
	if got := numberField(t, items[0].(map[string]interface{}), "id"); uint(got) != item.ID {
		t.Errorf("expected the first item to be the seeded one, got %v", got)
	}

	// An order without line items omits the array rather than sending null.
	bare := testOrder{
		UserID: clara.ID, OrderNumber: "ORD-2002", Status: "pending",
		TotalAmount: 10, PaymentStatus: "pending",
	}
	if err := db.Create(&bare).Error; err != nil {
		t.Fatalf("creating order without items: %v", err)
	}
	status, raw = request(t, app, http.MethodGet, fmt.Sprintf("/api/v1/admin/orders/%d", bare.ID), "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body = decodeJSON(t, raw)
	if items, ok := body["items"]; ok {
		if array, isArray := items.([]interface{}); !isArray || len(array) != 0 {
			t.Errorf("expected the items to be omitted or empty, got %v", items)
		}
	}

	// Soft-deleted orders are invisible.
	if err := db.Delete(&order).Error; err != nil {
		t.Fatalf("soft deleting order: %v", err)
	}
	status, raw = request(t, app, http.MethodGet, fmt.Sprintf("/api/v1/admin/orders/%d", order.ID), "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
}

func TestUpdateOrderStatusValidatesAndLogs(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	token := adminToken(t, admin.ID)

	order, _ := seedOrder(t, db, clara, "ORD-3001", "pending", "pending", 89.5)
	orderPath := fmt.Sprintf("/api/v1/admin/orders/%d", order.ID)

	for _, tc := range []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{"malformed id", "/api/v1/admin/orders/abc", `{"status":"shipped"}`, fiber.StatusBadRequest},
		{"unknown order", "/api/v1/admin/orders/4242", `{"status":"shipped"}`, fiber.StatusNotFound},
		{"unknown status", orderPath, `{"status":"teleported"}`, fiber.StatusBadRequest},
		{"unknown payment status", orderPath, `{"payment_status":"bitcoin"}`, fiber.StatusBadRequest},
		{"malformed body", orderPath, `{"status":`, fiber.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPut, tc.path, tc.body, token)
			if status != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, status, raw)
			}
		})
	}

	status, raw := request(t, app, http.MethodPut, orderPath,
		`{"status":"shipped","payment_status":"paid","notes":"Left with the concierge"}`, token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "Order updated successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	var stored testOrder
	if err := db.First(&stored, order.ID).Error; err != nil {
		t.Fatalf("loading order: %v", err)
	}
	if stored.Status != "shipped" || stored.PaymentStatus != "paid" {
		t.Errorf("expected the statuses to be updated, got %+v", stored)
	}
	if stored.Notes != "Left with the concierge" {
		t.Errorf("expected the note to be stored, got %q", stored.Notes)
	}

	// A body with only a note leaves the statuses untouched.
	status, raw = request(t, app, http.MethodPut, orderPath, `{"notes":"Second attempt"}`, token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if err := db.First(&stored, order.ID).Error; err != nil {
		t.Fatalf("loading order: %v", err)
	}
	if stored.Status != "shipped" || stored.PaymentStatus != "paid" {
		t.Errorf("expected the statuses to be preserved, got %+v", stored)
	}
	if stored.Notes != "Second attempt" {
		t.Errorf("expected the note to be replaced, got %q", stored.Notes)
	}

	var log models.ActivityLog
	if err := db.Where("resource = ? AND action = ?", "order", "updated").First(&log).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if log.ResourceID != order.ID || !strings.Contains(log.Description, "ORD-3001") {
		t.Errorf("unexpected audit log entry %+v", log)
	}
}

func TestDeleteOrderSoftDeletesAndHidesIt(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	token := adminToken(t, admin.ID)

	order, item := seedOrder(t, db, clara, "ORD-4001", "delivered", "paid", 150)

	for _, tc := range []struct {
		name   string
		path   string
		status int
	}{
		{"malformed id", "/api/v1/admin/orders/abc", fiber.StatusBadRequest},
		{"unknown order", "/api/v1/admin/orders/4242", fiber.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodDelete, tc.path, "", token)
			if status != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, status, raw)
			}
		})
	}

	status, raw := request(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/orders/%d", order.ID), "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "Order deleted successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	var live int64
	if err := db.Table("orders").Where("id = ? AND deleted_at IS NULL", order.ID).Count(&live).Error; err != nil {
		t.Fatalf("counting orders: %v", err)
	}
	if live != 0 {
		t.Errorf("expected the order to be soft deleted, got %d live rows", live)
	}

	// The line items survive so the financial history stays intact.
	var survivingItem testOrderItem
	if err := db.First(&survivingItem, item.ID).Error; err != nil {
		t.Fatalf("expected the order item to survive: %v", err)
	}

	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/orders", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 0 {
		t.Errorf("expected no orders in the list, got %v", total)
	}
	if orders := sliceField(t, body, "orders"); len(orders) != 0 {
		t.Errorf("expected an empty array, got %v", orders)
	}

	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/orders/analytics", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	analytics := objectField(t, decodeJSON(t, raw), "analytics")
	if total := numberField(t, analytics, "total_orders"); total != 0 {
		t.Errorf("expected the deleted order to be excluded from analytics, got %v", total)
	}
	if revenue := numberField(t, analytics, "total_revenue"); revenue != 0 {
		t.Errorf("expected no revenue, got %v", revenue)
	}

	var log models.ActivityLog
	if err := db.Where("resource = ? AND action = ?", "order", "deleted").First(&log).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if !strings.Contains(log.Description, "ORD-4001") {
		t.Errorf("expected the order number in the log, got %q", log.Description)
	}
}

func TestGetOrderAnalyticsAggregatesStatusesRevenueAndWindows(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	token := adminToken(t, admin.ID)

	now := time.Now()
	seedOrderAt(t, db, clara, "ORD-5001", "pending", "pending", 100, now)
	seedOrderAt(t, db, clara, "ORD-5002", "delivered", "paid", 200, now.AddDate(0, 0, -3))
	seedOrderAt(t, db, clara, "ORD-5003", "cancelled", "failed", 50, now.AddDate(0, 0, -40))

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/orders/analytics", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	analytics := objectField(t, decodeJSON(t, raw), "analytics")
	for field, want := range map[string]float64{
		"total_orders":      3,
		"pending_orders":    1,
		"processing_orders": 0,
		"shipped_orders":    0,
		"delivered_orders":  1,
		"cancelled_orders":  1,
		"total_revenue":     350,
		"pending_revenue":   100,
		"paid_revenue":      200,
		"orders_today":      1,
		"orders_this_week":  2,
		"orders_this_month": 2,
	} {
		if got := numberField(t, analytics, field); got != want {
			t.Errorf("expected %s to be %v, got %v", field, want, got)
		}
	}

	// Soft-deleted orders drop out of every aggregate.
	var first testOrder
	if err := db.Where("order_number = ?", "ORD-5002").First(&first).Error; err != nil {
		t.Fatalf("loading order: %v", err)
	}
	if err := db.Delete(&first).Error; err != nil {
		t.Fatalf("soft deleting order: %v", err)
	}

	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/orders/analytics", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	analytics = objectField(t, decodeJSON(t, raw), "analytics")
	if total := numberField(t, analytics, "total_orders"); total != 2 {
		t.Errorf("expected 2 orders after the delete, got %v", total)
	}
	if revenue := numberField(t, analytics, "paid_revenue"); revenue != 0 {
		t.Errorf("expected the paid revenue to be 0, got %v", revenue)
	}
	if week := numberField(t, analytics, "orders_this_week"); week != 1 {
		t.Errorf("expected 1 order this week after the delete, got %v", week)
	}
}

// TestGetOrderAnalyticsOnAnEmptyDatabase pins the zero value shape the
// dashboard renders before any order exists.
func TestGetOrderAnalyticsOnAnEmptyDatabase(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/orders/analytics", "", adminToken(t, 1))
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	analytics := objectField(t, decodeJSON(t, raw), "analytics")
	for field, want := range map[string]float64{
		"total_orders":      0,
		"total_revenue":     0,
		"pending_revenue":   0,
		"paid_revenue":      0,
		"orders_today":      0,
		"orders_this_week":  0,
		"orders_this_month": 0,
	} {
		if got := numberField(t, analytics, field); got != want {
			t.Errorf("expected %s to be %v, got %v", field, want, got)
		}
	}
}
