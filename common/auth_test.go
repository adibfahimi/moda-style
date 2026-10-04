package common

import (
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// newAuthApp builds a Fiber app whose /private route is protected by
// RequireAuth and echoes the values the middleware stored in Locals.
func newAuthApp() *fiber.App {
	app := fiber.New()
	app.Get("/private", RequireAuth, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"userID":   c.Locals("userID"),
			"email":    c.Locals("email"),
			"userName": c.Locals("userName"),
			"role":     c.Locals("role"),
		})
	})
	return app
}

func TestGenerateToken_CarriesIdentityAndExpiry(t *testing.T) {
	token, err := GenerateToken(42, "ada@example.com", "Ada Lovelace", "admin")
	if err != nil {
		t.Fatalf("GenerateToken returned an error: %v", err)
	}

	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(_ *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil {
		t.Fatalf("generated token could not be parsed: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("generated token should be valid")
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		t.Fatalf("unexpected claims type %T", parsed.Claims)
	}

	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Email != "ada@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "ada@example.com")
	}
	if claims.UserName != "Ada Lovelace" {
		t.Errorf("UserName = %q, want %q", claims.UserName, "Ada Lovelace")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}

	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt should be set")
	}
	if ttl := time.Until(claims.ExpiresAt.Time); ttl <= 0 || ttl > TokenTTL+time.Minute {
		t.Errorf("token TTL = %v, want within (0, %v]", ttl, TokenTTL+time.Minute)
	}
	if claims.IssuedAt == nil {
		t.Fatal("IssuedAt should be set")
	}
}

func TestGenerateToken_UsesHS256(t *testing.T) {
	token, err := GenerateToken(1, "a@b.com", "A", "customer")
	if err != nil {
		t.Fatalf("GenerateToken returned an error: %v", err)
	}

	parsed, _, err := jwt.NewParser().ParseUnverified(token, &Claims{})
	if err != nil {
		t.Fatalf("ParseUnverified failed: %v", err)
	}
	if got := parsed.Method.Alg(); got != signingAlgorithm {
		t.Errorf("signing algorithm = %q, want %q", got, signingAlgorithm)
	}
}
