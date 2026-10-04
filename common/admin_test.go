package common

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// newAdminApp builds an app whose /admin route is guarded by the full
// RequireAuth + RequireAdmin chain, mirroring how services register admin
// routes.
func newAdminApp() *fiber.App {
	app := fiber.New()
	app.Get("/admin", RequireAuth, RequireAdmin, func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func TestRequireAdmin_EnforcesAuthenticationAndRole(t *testing.T) {
	adminToken, err := GenerateToken(1, "root@example.com", "Root", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	customerToken, err := GenerateToken(2, "shopper@example.com", "Shopper", "customer")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		wantStatus int
		wantError  string
	}{
		{name: "no token", token: "", wantStatus: http.StatusUnauthorized, wantError: "Authorization header required"},
		{name: "non-admin role", token: customerToken, wantStatus: http.StatusForbidden, wantError: "Admin access required"},
		{name: "admin role", token: adminToken, wantStatus: http.StatusOK},
	}

	app := newAdminApp()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			headers := map[string]string{}
			if tc.token != "" {
				headers["Authorization"] = "Bearer " + tc.token
			}

			resp, raw := performRequest(t, app, http.MethodGet, "/admin", "", headers)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", resp.StatusCode, tc.wantStatus, raw)
			}

			if tc.wantError == "" {
				if string(raw) != "ok" {
					t.Errorf("body = %q, want %q", string(raw), "ok")
				}
				return
			}
			if got := errorMessage(t, raw); got != tc.wantError {
				t.Errorf("error = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestRequireAdmin_RejectsUnauthenticatedRequestWhenUsedAlone(t *testing.T) {
	app := fiber.New()
	app.Get("/admin", RequireAdmin, func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	resp, raw := performRequest(t, app, http.MethodGet, "/admin", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if got := errorMessage(t, raw); got != "User not authenticated" {
		t.Errorf("error = %q, want %q", got, "User not authenticated")
	}
}
