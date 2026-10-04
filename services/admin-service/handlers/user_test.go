package handlers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/adibfahimi/moda-style/services/admin-service/models"
	"github.com/gofiber/fiber/v2"
)

func TestUserRoutesRequireTheAdminRole(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")

	routes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/admin/users", ""},
		{http.MethodGet, "/api/v1/admin/users/analytics", ""},
		{http.MethodGet, fmt.Sprintf("/api/v1/admin/users/%d", admin.ID), ""},
		{http.MethodPut, fmt.Sprintf("/api/v1/admin/users/%d", admin.ID), `{"name":"Ada","email":"ada@example.com","role":"admin"}`},
		{http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%d", admin.ID), ""},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/ban", admin.ID), ""},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/unban", admin.ID), ""},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			status, raw := request(t, app, route.method, route.path, route.body, "")
			if status != fiber.StatusUnauthorized {
				t.Fatalf("expected 401 without a token, got %d: %s", status, raw)
			}

			status, raw = request(t, app, route.method, route.path, route.body, tokenFor(t, 99, "user"))
			if status != fiber.StatusForbidden {
				t.Fatalf("expected 403 for a non-admin, got %d: %s", status, raw)
			}
		})
	}
}

func TestGetUsersFiltersAndPaginates(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	seedUser(t, db, "Bella Buyer", "bella@example.com", "user")

	_, dress := seedCatalogue(t, db, "Silk Midi Dress", 12)
	seedReview(t, db, clara, dress, 5)
	seedReview(t, db, clara, dress, 4)
	if err := db.Create(&testWishlistItem{UserID: clara.ID, ProductID: dress.ID}).Error; err != nil {
		t.Fatalf("creating wishlist item: %v", err)
	}

	token := adminToken(t, admin.ID)

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/users", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 3 {
		t.Errorf("expected 3 users in total, got %v", total)
	}
	if page := numberField(t, body, "page"); page != 1 {
		t.Errorf("expected page 1, got %v", page)
	}
	if limit := numberField(t, body, "limit"); limit != 20 {
		t.Errorf("expected the default limit 20, got %v", limit)
	}

	users := sliceField(t, body, "users")
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d: %s", len(users), raw)
	}

	byEmail := make(map[string]map[string]interface{}, len(users))
	for _, entry := range users {
		user := entry.(map[string]interface{})
		byEmail[stringField(t, user, "email")] = user
	}

	claraStats, ok := byEmail["clara@example.com"]
	if !ok {
		t.Fatalf("expected Clara in the list, got %s", raw)
	}
	if count := numberField(t, claraStats, "review_count"); count != 2 {
		t.Errorf("expected 2 reviews for Clara, got %v", count)
	}
	if count := numberField(t, claraStats, "wishlist_count"); count != 1 {
		t.Errorf("expected 1 wishlist item for Clara, got %v", count)
	}
	if banned, _ := claraStats["banned"].(bool); banned {
		t.Errorf("expected Clara not to be banned, got %v", claraStats)
	}

	for _, tc := range []struct {
		name  string
		query string
		want  float64
	}{
		{"filter by role", "?role=user", 2},
		{"filter by admin role", "?role=admin", 1},
		{"search is case insensitive", "?search=CLARA", 1},
		{"search matches the email domain", "?search=example.com", 3},
		{"search without matches", "?search=nobody", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, "/api/v1/admin/users"+tc.query, "", token)
			if status != fiber.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", status, raw)
			}
			body := decodeJSON(t, raw)
			if total := numberField(t, body, "total"); total != tc.want {
				t.Errorf("expected total %v, got %v", tc.want, total)
			}
			if got := float64(len(sliceField(t, body, "users"))); got != tc.want {
				t.Errorf("expected %v users, got %v", tc.want, got)
			}
		})
	}

	// Second page of two.
	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/users?page=2&limit=2", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	body = decodeJSON(t, raw)
	if page := numberField(t, body, "page"); page != 2 {
		t.Errorf("expected page 2, got %v", page)
	}
	if users := sliceField(t, body, "users"); len(users) != 1 {
		t.Errorf("expected 1 user on the second page, got %d", len(users))
	}

	// The page size is capped at 100.
	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/users?limit=500", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if limit := numberField(t, decodeJSON(t, raw), "limit"); limit != 100 {
		t.Errorf("expected the limit to be capped at 100, got %v", limit)
	}
}

func TestGetUserDetailsReturnsCountsAndReviews(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")

	_, dress := seedCatalogue(t, db, "Silk Midi Dress", 12)
	seedReview(t, db, clara, dress, 5)
	seedReview(t, db, clara, dress, 3)
	if err := db.Create(&testWishlistItem{UserID: clara.ID, ProductID: dress.ID}).Error; err != nil {
		t.Fatalf("creating wishlist item: %v", err)
	}

	token := adminToken(t, admin.ID)
	path := fmt.Sprintf("/api/v1/admin/users/%d", clara.ID)

	status, raw := request(t, app, http.MethodGet, path, "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	user := objectField(t, body, "user")
	if name := stringField(t, user, "name"); name != "Clara Client" {
		t.Errorf("expected Clara, got %q", name)
	}
	if role := stringField(t, user, "role"); role != "user" {
		t.Errorf("expected the user role, got %q", role)
	}
	if count := numberField(t, user, "review_count"); count != 2 {
		t.Errorf("expected 2 reviews, got %v", count)
	}
	if count := numberField(t, user, "wishlist_count"); count != 1 {
		t.Errorf("expected 1 wishlist item, got %v", count)
	}

	reviews := sliceField(t, body, "reviews")
	if len(reviews) != 2 {
		t.Fatalf("expected 2 reviews, got %d: %s", len(reviews), raw)
	}
	first := reviews[0].(map[string]interface{})
	if productName := stringField(t, first, "product_name"); productName != "Silk Midi Dress" {
		t.Errorf("expected the product name from the join, got %q", productName)
	}
	if rating := numberField(t, first, "rating"); rating != 3 && rating != 5 {
		t.Errorf("unexpected rating %v", rating)
	}

	for _, tc := range []struct {
		name   string
		path   string
		status int
	}{
		{"invalid id", "/api/v1/admin/users/not-a-number", fiber.StatusBadRequest},
		{"unknown user", "/api/v1/admin/users/4242", fiber.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, tc.path, "", token)
			if status != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, status, raw)
			}
		})
	}
}

func TestUpdateUserValidatesAndGuards(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	seedUser(t, db, "Bella Buyer", "bella@example.com", "user")

	token := adminToken(t, admin.ID)
	path := fmt.Sprintf("/api/v1/admin/users/%d", clara.ID)

	// Validation failures: empty body, short name and an unknown role.
	for _, tc := range []struct {
		name string
		body string
	}{
		{"empty body", `{}`},
		{"short name", `{"name":"A","email":"clara@example.com","role":"user"}`},
		{"invalid email", `{"name":"Clara","email":"not-an-email","role":"user"}`},
		{"unknown role", `{"name":"Clara","email":"clara@example.com","role":"manager"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPut, path, tc.body, token)
			if status != fiber.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
			}
			if msg := stringField(t, decodeJSON(t, raw), "error"); msg == "" {
				t.Errorf("expected a validation message, got %s", raw)
			}
		})
	}

	// Unknown user.
	status, raw := request(t, app, http.MethodPut, "/api/v1/admin/users/4242",
		`{"name":"Clara","email":"clara@example.com","role":"user"}`, token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}

	// Conflict with another account's email.
	status, raw = request(t, app, http.MethodPut, path,
		`{"name":"Clara","email":"ada@example.com","role":"user"}`, token)
	if status != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Email already in use" {
		t.Errorf("unexpected error message %q", msg)
	}

	// Successful update, keeping the existing email.
	status, raw = request(t, app, http.MethodPut, path,
		`{"name":"Clara Couture","email":"clara@example.com","role":"admin"}`, token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "User updated successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	var updated testUser
	if err := db.First(&updated, clara.ID).Error; err != nil {
		t.Fatalf("loading user: %v", err)
	}
	if updated.Name != "Clara Couture" {
		t.Errorf("expected the name to be updated, got %q", updated.Name)
	}
	if updated.Role != "admin" {
		t.Errorf("expected the role to be updated, got %q", updated.Role)
	}
	if updated.Email != "clara@example.com" {
		t.Errorf("expected the email to be kept, got %q", updated.Email)
	}

	// The change is written to the audit trail.
	var log models.ActivityLog
	if err := db.Where("resource = ? AND action = ?", "user", "updated").First(&log).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if log.AdminID != admin.ID {
		t.Errorf("expected the acting admin %d, got %d", admin.ID, log.AdminID)
	}
	if log.ResourceID != clara.ID {
		t.Errorf("expected the target user %d, got %d", clara.ID, log.ResourceID)
	}
}

func TestDeleteUserSoftDeletesAndHidesTheAccount(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")

	token := adminToken(t, admin.ID)

	// An admin cannot delete their own account.
	status, raw := request(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%d", admin.ID), "", token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Cannot delete your own account" {
		t.Errorf("unexpected error message %q", msg)
	}

	// Unknown and malformed ids are rejected instead of silently succeeding.
	status, raw = request(t, app, http.MethodDelete, "/api/v1/admin/users/4242", "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
	status, raw = request(t, app, http.MethodDelete, "/api/v1/admin/users/not-a-number", "", token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}

	status, raw = request(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%d", clara.ID), "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "User deleted successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	// The row is soft deleted, not removed.
	var softDeleted int64
	if err := db.Table("users").
		Where("id = ? AND deleted_at IS NOT NULL", clara.ID).
		Count(&softDeleted).Error; err != nil {
		t.Fatalf("counting deleted users: %v", err)
	}
	if softDeleted != 1 {
		t.Errorf("expected the user to be soft deleted, got %d rows", softDeleted)
	}

	// Deleted accounts disappear from the admin listing.
	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/users", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if total := numberField(t, decodeJSON(t, raw), "total"); total != 1 {
		t.Errorf("expected only the admin to remain, got %v", total)
	}

	var log models.ActivityLog
	if err := db.Where("action = ?", "deleted").First(&log).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if log.ResourceID != clara.ID {
		t.Errorf("expected the deleted user %d in the audit log, got %d", clara.ID, log.ResourceID)
	}
	if !strings.Contains(log.Description, "Clara Client") {
		t.Errorf("expected the user name in the audit description, got %q", log.Description)
	}
}

func TestBanAndUnbanUser(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")

	token := adminToken(t, admin.ID)
	banPath := fmt.Sprintf("/api/v1/admin/users/%d/ban", clara.ID)
	unbanPath := fmt.Sprintf("/api/v1/admin/users/%d/unban", clara.ID)

	// Self-banning and unknown accounts are rejected.
	status, raw := request(t, app, http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/ban", admin.ID), "", token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for a self-ban, got %d: %s", status, raw)
	}
	status, raw = request(t, app, http.MethodPost, "/api/v1/admin/users/4242/ban", "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found for a missing user, got %d: %s", status, raw)
	}
	status, raw = request(t, app, http.MethodPost, "/api/v1/admin/users/not-a-number/ban", "", token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for a malformed id, got %d: %s", status, raw)
	}

	// Banning flips the flag and is idempotent-safe.
	status, raw = request(t, app, http.MethodPost, banPath, "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "User banned successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	var banned testUser
	if err := db.First(&banned, clara.ID).Error; err != nil {
		t.Fatalf("loading user: %v", err)
	}
	if !banned.Banned {
		t.Error("expected the user to be flagged as banned")
	}

	status, raw = request(t, app, http.MethodPost, banPath, "", token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when banning twice, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "User is already banned" {
		t.Errorf("unexpected error message %q", msg)
	}

	// The ban shows up in the audit trail and in the user listing.
	var banLog models.ActivityLog
	if err := db.Where("action = ? AND resource = ?", "banned", "user").First(&banLog).Error; err != nil {
		t.Fatalf("expected an audit log entry: %v", err)
	}
	if banLog.ResourceID != clara.ID || banLog.AdminID != admin.ID {
		t.Errorf("unexpected audit log %+v", banLog)
	}
	if !strings.Contains(banLog.Description, "Clara Client") {
		t.Errorf("unexpected audit description %q", banLog.Description)
	}

	status, raw = request(t, app, http.MethodGet, "/api/v1/admin/users?search=clara", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	users := sliceField(t, decodeJSON(t, raw), "users")
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d: %s", len(users), raw)
	}
	if banned, _ := users[0].(map[string]interface{})["banned"].(bool); !banned {
		t.Errorf("expected the listing to expose the ban, got %s", raw)
	}

	// Unbanning restores the account, but only when it is actually banned.
	status, raw = request(t, app, http.MethodPost, unbanPath, "", token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != "User unbanned successfully" {
		t.Errorf("unexpected message %q", msg)
	}

	var unbanned testUser
	if err := db.First(&unbanned, clara.ID).Error; err != nil {
		t.Fatalf("loading user: %v", err)
	}
	if unbanned.Banned {
		t.Error("expected the ban to be lifted")
	}

	status, raw = request(t, app, http.MethodPost, unbanPath, "", token)
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when unbanning twice, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "User is not banned" {
		t.Errorf("unexpected error message %q", msg)
	}

	status, raw = request(t, app, http.MethodPost, "/api/v1/admin/users/4242/unban", "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
}

func TestGetUserAnalyticsCountsRolesAndActivity(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	admin := seedUser(t, db, "Ada Admin", "ada@example.com", "admin")
	clara := seedUser(t, db, "Clara Client", "clara@example.com", "user")
	bob := seedUser(t, db, "Bob Buyer", "bob@example.com", "user")

	// Only users with at least one live review count as active reviewers.
	_, dress := seedCatalogue(t, db, "Silk Midi Dress", 5)
	seedReview(t, db, clara, dress, 5)
	seedReview(t, db, bob, dress, 3)

	// A soft-deleted account is invisible to every counter.
	dana := seedUser(t, db, "Dana Dropped", "dana@example.com", "user")
	if err := db.Delete(&dana).Error; err != nil {
		t.Fatalf("soft deleting user: %v", err)
	}

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/users/analytics", "", adminToken(t, admin.ID))
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	analytics := objectField(t, decodeJSON(t, raw), "analytics")
	for field, want := range map[string]float64{
		"total_users":         3,
		"admin_users":         1,
		"regular_users":       2,
		"new_users_today":     3,
		"new_users_this_week": 3,
		"active_reviewers":    2,
	} {
		if got := numberField(t, analytics, field); got != want {
			t.Errorf("expected %s to be %v, got %v", field, want, got)
		}
	}
}

func TestGetUserAnalyticsOnAnEmptyDatabase(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	status, raw := request(t, app, http.MethodGet, "/api/v1/admin/users/analytics", "", adminToken(t, 1))
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	analytics := objectField(t, decodeJSON(t, raw), "analytics")
	for field := range map[string]struct{}{
		"total_users": {}, "admin_users": {}, "regular_users": {},
		"new_users_today": {}, "new_users_this_week": {}, "active_reviewers": {},
	} {
		if got := numberField(t, analytics, field); got != 0 {
			t.Errorf("expected %s to be 0, got %v", field, got)
		}
	}
}
