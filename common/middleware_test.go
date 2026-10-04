package common

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// signToken builds and signs a token with the given method, secret and claims.
func signToken(t *testing.T, method jwt.SigningMethod, secret string, claims Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signing token failed: %v", err)
	}
	return signed
}

func TestRequireAuth_RejectsMalformedRequests(t *testing.T) {
	validToken, err := GenerateToken(7, "grace@example.com", "Grace", "customer")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	tests := []struct {
		name       string
		authHeader string
		wantError  string
	}{
		{name: "missing header", authHeader: "", wantError: "Authorization header required"},
		{name: "wrong scheme", authHeader: "Token " + validToken, wantError: "Invalid authorization format"},
		{name: "missing token", authHeader: "Bearer", wantError: "Invalid authorization format"},
		{name: "too many parts", authHeader: "Bearer a b", wantError: "Invalid authorization format"},
		{name: "garbage token", authHeader: "Bearer not-a-jwt", wantError: "Invalid or expired token"},
	}

	app := newAuthApp()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			headers := map[string]string{}
			if tc.authHeader != "" {
				headers["Authorization"] = tc.authHeader
			}

			resp, raw := performRequest(t, app, http.MethodGet, "/private", "", headers)
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
			}
			if got := errorMessage(t, raw); got != tc.wantError {
				t.Errorf("error = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestRequireAuth_RejectsTokensItCannotTrust(t *testing.T) {
	future := jwt.NewNumericDate(time.Now().Add(time.Hour))
	past := jwt.NewNumericDate(time.Now().Add(-time.Minute))

	tests := []struct {
		name   string
		method jwt.SigningMethod
		secret string
		claims Claims
	}{
		{
			name:   "signed with a different secret",
			method: jwt.SigningMethodHS256,
			secret: "not-the-configured-secret",
			claims: Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: future}},
		},
		{
			name:   "disallowed algorithm",
			method: jwt.SigningMethodHS512,
			secret: testJWTSecret,
			claims: Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: future}},
		},
		{
			name:   "expired token",
			method: jwt.SigningMethodHS256,
			secret: testJWTSecret,
			claims: Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: past}},
		},
	}

	app := newAuthApp()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			signed := signToken(t, tc.method, tc.secret, tc.claims)

			resp, raw := performRequest(t, app, http.MethodGet, "/private", "", map[string]string{
				"Authorization": "Bearer " + signed,
			})
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
			}
			if got := errorMessage(t, raw); got != "Invalid or expired token" {
				t.Errorf("error = %q, want %q", got, "Invalid or expired token")
			}
		})
	}
}

func TestRequireAuth_PopulatesLocalsForValidToken(t *testing.T) {
	token, err := GenerateToken(99, "alan@example.com", "Alan Turing", "admin")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	resp, raw := performRequest(t, newAuthApp(), http.MethodGet, "/private", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", resp.StatusCode, http.StatusOK, raw)
	}

	body := decodeJSON(t, raw)
	checks := map[string]interface{}{
		"userID":   float64(99),
		"email":    "alan@example.com",
		"userName": "Alan Turing",
		"role":     "admin",
	}
	for field, want := range checks {
		if got := body[field]; got != want {
			t.Errorf("locals[%q] = %v, want %v", field, got, want)
		}
	}
}
