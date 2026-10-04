package handlers_test

import (
	"encoding/hex"
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
	"github.com/adibfahimi/moda-style/services/auth-service/database"
	"github.com/adibfahimi/moda-style/services/auth-service/handlers"
	"github.com/adibfahimi/moda-style/services/auth-service/models"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testJWTSecret is the signing key every test in this package uses. TestMain
// installs it before any test runs because common resolves JWT_SECRET lazily on
// the first signing or verification operation.
const testJWTSecret = "auth-service-test-secret"

// dbCounter hands every test its own in-memory database so tests never observe
// each other's rows.
var dbCounter atomic.Uint64

func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

// newTestDB returns an isolated in-memory SQLite database with the auth schema
// migrated, and points the package-level handler handle (database.DB) at it.
//
// SQLite is used instead of PostgreSQL so the handler tests run anywhere: the
// SQL exercised here is dialect neutral. The pool is capped at a single
// connection and closed on cleanup, which also destroys the in-memory database.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:auth_test_%d?mode=memory&cache=shared", dbCounter.Add(1))
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

	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrating auth schema: %v", err)
	}

	database.DB = db
	return db
}

// newTestApp mounts the same routes as services/auth-service/main.go (minus the
// CORS and health-check middleware) so tests exercise the real Fiber call path,
// including common.RequireAuth on the profile routes.
func newTestApp() *fiber.App {
	app := fiber.New()

	auth := app.Group("/api/v1/auth")
	auth.Post("/register", handlers.Register)
	auth.Post("/login", handlers.Login)
	auth.Get("/profile", common.RequireAuth, handlers.GetProfile)
	auth.Patch("/profile", common.RequireAuth, handlers.UpdateProfile)
	auth.Post("/reset-password", handlers.ResetPassword)

	return app
}

// request drives app with an HTTP request and returns the status code and the
// raw JSON body. A non-empty token is sent as a bearer credential.
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

// mapField returns the named nested object, failing the test when it is missing.
func mapField(t *testing.T, body map[string]interface{}, key string) map[string]interface{} {
	t.Helper()

	value, ok := body[key].(map[string]interface{})
	if !ok {
		t.Fatalf("field %q is missing or not an object in %v", key, body)
	}
	return value
}

// register creates an account through the public registration endpoint and
// returns the issued access token, failing the test when registration fails.
func register(t *testing.T, app *fiber.App, name, email, password string) string {
	t.Helper()

	payload := fmt.Sprintf(`{"name":%q,"email":%q,"password":%q}`, name, email, password)
	status, raw := request(t, app, http.MethodPost, "/api/v1/auth/register", payload, "")
	if status != fiber.StatusCreated {
		t.Fatalf("registering %s failed with %d: %s", email, status, raw)
	}

	return stringField(t, decodeJSON(t, raw), "token")
}

func TestRegisterCreatesUserWithHashedPassword(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	status, raw := request(t, app, http.MethodPost, "/api/v1/auth/register",
		`{"name":"Ada Lovelace","email":"ada@example.com","password":"secret123"}`, "")

	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Registration successful" {
		t.Errorf("unexpected message %q", msg)
	}
	if token := stringField(t, body, "token"); token == "" {
		t.Error("expected a non-empty token")
	}

	user := mapField(t, body, "user")
	if got := stringField(t, user, "email"); got != "ada@example.com" {
		t.Errorf("unexpected email %q", got)
	}
	if got := stringField(t, user, "role"); got != "user" {
		t.Errorf("expected new accounts to default to role \"user\", got %q", got)
	}

	// The password must never be stored in plaintext.
	var stored models.User
	if err := db.Where("email = ?", "ada@example.com").First(&stored).Error; err != nil {
		t.Fatalf("loading created user: %v", err)
	}
	if stored.Password == "secret123" {
		t.Fatal("password was stored in plaintext")
	}
	if !stored.CheckPassword("secret123") {
		t.Error("stored bcrypt hash does not match the submitted password")
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	payload := `{"name":"Ada Lovelace","email":"ada@example.com","password":"secret123"}`
	if status, raw := request(t, app, http.MethodPost, "/api/v1/auth/register", payload, ""); status != fiber.StatusCreated {
		t.Fatalf("first registration failed with %d: %s", status, raw)
	}

	status, raw := request(t, app, http.MethodPost, "/api/v1/auth/register", payload, "")
	if status != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Email already registered" {
		t.Errorf("unexpected error message %q", msg)
	}
}

func TestRegisterValidatesRequestBody(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	tests := []struct {
		name        string
		body        string
		wantMessage string
	}{
		{"malformed json", `{"email":`, "Invalid request body"},
		{"missing email", `{"name":"Ada Lovelace","password":"secret123"}`, "email is required"},
		{"invalid email", `{"name":"Ada Lovelace","email":"not-an-email","password":"secret123"}`, "email must be a valid email"},
		{"missing password", `{"name":"Ada Lovelace","email":"ada@example.com"}`, "password is required"},
		{"short password", `{"name":"Ada Lovelace","email":"ada@example.com","password":"123"}`, "password must be at least 6 characters"},
		{"missing name", `{"email":"ada@example.com","password":"secret123"}`, "name is required"},
		{"short name", `{"name":"A","email":"ada@example.com","password":"secret123"}`, "name must be at least 2 characters"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, "/api/v1/auth/register", tc.body, "")
			if status != fiber.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
			}
			if msg := stringField(t, decodeJSON(t, raw), "error"); msg != tc.wantMessage {
				t.Errorf("expected error %q, got %q", tc.wantMessage, msg)
			}
		})
	}
}

func TestLoginReturnsTokenThatAuthenticatesProfileRequest(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	register(t, app, "Grace Hopper", "grace@example.com", "secret123")

	status, raw := request(t, app, http.MethodPost, "/api/v1/auth/login",
		`{"email":"grace@example.com","password":"secret123"}`, "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Login successful" {
		t.Errorf("unexpected message %q", msg)
	}
	token := stringField(t, body, "token")

	// The token issued by /login must be accepted by the protected profile route.
	status, raw = request(t, app, http.MethodGet, "/api/v1/auth/profile", "", token)
	if status != fiber.StatusOK {
		t.Fatalf("profile request with login token failed with %d: %s", status, raw)
	}
	if email := stringField(t, mapField(t, decodeJSON(t, raw), "user"), "email"); email != "grace@example.com" {
		t.Errorf("profile returned the wrong user %q", email)
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	register(t, app, "Grace Hopper", "grace@example.com", "secret123")

	tests := []struct {
		name string
		body string
	}{
		{"unknown email", `{"email":"nobody@example.com","password":"secret123"}`},
		{"wrong password", `{"email":"grace@example.com","password":"wrong-password"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, "/api/v1/auth/login", tc.body, "")
			if status != fiber.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized, got %d: %s", status, raw)
			}
			// The message must not reveal which of the two credentials was wrong.
			if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Invalid email or password" {
				t.Errorf("unexpected error message %q", msg)
			}
		})
	}
}

func TestLoginValidatesRequestBody(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	status, raw := request(t, app, http.MethodPost, "/api/v1/auth/login", `{"email":"grace@example.com"}`, "")
	if status != fiber.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "password is required" {
		t.Errorf("unexpected error message %q", msg)
	}
}

func TestGetProfileRejectsUnauthenticatedRequests(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	tests := []struct {
		name        string
		token       string
		wantMessage string
	}{
		{"missing header", "", "Authorization header required"},
		{"malformed token", "not-a-jwt", "Invalid or expired token"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodGet, "/api/v1/auth/profile", "", tc.token)
			if status != fiber.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized, got %d: %s", status, raw)
			}
			if msg := stringField(t, decodeJSON(t, raw), "error"); msg != tc.wantMessage {
				t.Errorf("expected error %q, got %q", tc.wantMessage, msg)
			}
		})
	}
}

func TestGetProfileReturnsNotFoundWhenUserWasDeleted(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	token := register(t, app, "Grace Hopper", "grace@example.com", "secret123")

	// Hard delete keeps the token valid while removing the account, which is the
	// only way to reach the 404 branch of GetProfile.
	if err := db.Unscoped().Where("email = ?", "grace@example.com").Delete(&models.User{}).Error; err != nil {
		t.Fatalf("deleting user: %v", err)
	}

	status, raw := request(t, app, http.MethodGet, "/api/v1/auth/profile", "", token)
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
}

func TestUpdateProfileChangesNameAndEmail(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	token := register(t, app, "Grace Hopper", "grace@example.com", "secret123")

	status, raw := request(t, app, http.MethodPatch, "/api/v1/auth/profile",
		`{"name":"Rear Admiral Grace Hopper","email":"grace.hopper@example.com"}`, token)
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Profile updated successfully" {
		t.Errorf("unexpected message %q", msg)
	}
	user := mapField(t, body, "user")
	if got := stringField(t, user, "name"); got != "Rear Admiral Grace Hopper" {
		t.Errorf("unexpected name %q", got)
	}

	var stored models.User
	if err := db.Where("id = ?", user["id"]).First(&stored).Error; err != nil {
		t.Fatalf("reloading user: %v", err)
	}
	if stored.Email != "grace.hopper@example.com" {
		t.Errorf("email was not persisted, got %q", stored.Email)
	}
	if !stored.CheckPassword("secret123") {
		t.Error("updating the profile must not touch the password hash")
	}
}

func TestUpdateProfileRejectsEmailOfAnotherUser(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	register(t, app, "Grace Hopper", "grace@example.com", "secret123")
	token := register(t, app, "Ada Lovelace", "ada@example.com", "secret123")

	status, raw := request(t, app, http.MethodPatch, "/api/v1/auth/profile",
		`{"email":"grace@example.com"}`, token)
	if status != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Email already in use" {
		t.Errorf("unexpected error message %q", msg)
	}
}

func TestUpdateProfileValidatesRequestBody(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	token := register(t, app, "Grace Hopper", "grace@example.com", "secret123")

	tests := []struct {
		name        string
		body        string
		wantMessage string
	}{
		{"invalid email", `{"email":"not-an-email"}`, "email must be a valid email"},
		{"short name", `{"name":"G"}`, "name must be at least 2 characters"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPatch, "/api/v1/auth/profile", tc.body, token)
			if status != fiber.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
			}
			if msg := stringField(t, decodeJSON(t, raw), "error"); msg != tc.wantMessage {
				t.Errorf("expected error %q, got %q", tc.wantMessage, msg)
			}
		})
	}
}

func TestResetPasswordStoresOneHourTokenForKnownEmail(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()

	register(t, app, "Grace Hopper", "grace@example.com", "secret123")

	status, raw := request(t, app, http.MethodPost, "/api/v1/auth/reset-password",
		`{"email":"grace@example.com"}`, "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	var stored models.User
	if err := db.Where("email = ?", "grace@example.com").First(&stored).Error; err != nil {
		t.Fatalf("reloading user: %v", err)
	}
	if stored.ResetToken == nil {
		t.Fatal("expected a reset token to be stored")
	}
	if len(*stored.ResetToken) != 64 {
		t.Errorf("expected a 64 character hex token, got %q", *stored.ResetToken)
	}
	if _, err := hex.DecodeString(*stored.ResetToken); err != nil {
		t.Errorf("reset token is not valid hex: %v", err)
	}
	if stored.ResetTokenExpiry == nil {
		t.Fatal("expected a reset token expiry to be stored")
	}

	ttl := time.Until(*stored.ResetTokenExpiry)
	if ttl < 55*time.Minute || ttl > 65*time.Minute {
		t.Errorf("expected a one hour expiry, got %s", ttl)
	}
}

func TestResetPasswordDoesNotRevealUnknownEmails(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	want := "If the email exists, a password reset link has been sent"

	status, raw := request(t, app, http.MethodPost, "/api/v1/auth/reset-password",
		`{"email":"nobody@example.com"}`, "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "message"); msg != want {
		t.Errorf("expected message %q, got %q", want, msg)
	}
}
