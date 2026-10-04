// Package common holds the cross-cutting building blocks that every Moda Style
// microservice shares: JWT authentication middlewares, a PostgreSQL/GORM
// connection helper, request-validation utilities and role-based access
// control.
//
// Services import this package as
// github.com/adibfahimi/moda-style/common and rely on it to guarantee that
// authentication and error responses behave identically everywhere.
package common

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// TokenTTL is how long an issued JWT stays valid.
const TokenTTL = 24 * time.Hour

// signingAlgorithm is the JWT signing algorithm used for every token. It is
// fixed to HS256 so that verification can reject tokens that try to use the
// "none" algorithm or an unexpected method.
const signingAlgorithm = "HS256"

// jwtSecret caches the HMAC key read from the JWT_SECRET environment variable.
// It is resolved lazily and exactly once, so importing this package does not
// require the variable to be set — the process only fails when it actually
// tries to sign or verify a token.
var (
	jwtSecretOnce sync.Once
	jwtSecret     []byte
)

// getJWTSecret returns the process-wide JWT signing key, reading it from the
// JWT_SECRET environment variable the first time it is needed.
//
// The process terminates with a fatal error if JWT_SECRET is empty, because
// running without a signing key would let anyone forge access tokens.
func getJWTSecret() []byte {
	jwtSecretOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			log.Fatal("JWT_SECRET environment variable is required")
		}
		jwtSecret = []byte(secret)
	})
	return jwtSecret
}

// Claims is the set of JWT claims embedded in every Moda Style access token.
// It carries the minimum identity information a service needs to authorise a
// request without an extra database round-trip.
type Claims struct {
	// UserID is the primary key of the authenticated user.
	UserID uint `json:"user_id"`
	// Email is the user's email address at the time the token was issued.
	Email string `json:"email"`
	// UserName is the user's display name.
	UserName string `json:"user_name"`
	// Role is the user's coarse-grained role, e.g. "customer" or "admin".
	Role string `json:"role"`
	// RegisteredClaims adds the standard "exp" and "iat" timestamps.
	jwt.RegisteredClaims
}

// GenerateToken signs a new JWT for the given identity.
//
// The returned token is valid for TokenTTL and is signed with the key taken
// from the JWT_SECRET environment variable. It returns an error only if signing
// itself fails.
func GenerateToken(userID uint, email string, userName string, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Email:    email,
		UserName: userName,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// RequireAuth is Fiber middleware that authenticates a request from its
// "Authorization: Bearer <token>" header.
//
// On success it verifies the token and stores the claims in the request Locals
// under the keys "userID", "email", "userName" and "role" before invoking the
// next handler. On failure it writes a 401 JSON error and stops the chain.
func RequireAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return SendErrorResponse(c, fiber.StatusUnauthorized, "Authorization header required")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return SendErrorResponse(c, fiber.StatusUnauthorized, "Invalid authorization format")
	}

	tokenString := parts[1]

	// Parse and verify the token. WithValidMethods pins the algorithm to HS256
	// so that a token signed with a different method is rejected outright.
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(_ *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		},
		jwt.WithValidMethods([]string{signingAlgorithm}),
	)

	if err != nil || !token.Valid {
		return SendErrorResponse(c, fiber.StatusUnauthorized, "Invalid or expired token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return SendErrorResponse(c, fiber.StatusUnauthorized, "Invalid token claims")
	}

	c.Locals("userID", claims.UserID)
	c.Locals("email", claims.Email)
	c.Locals("userName", claims.UserName)
	c.Locals("role", claims.Role)

	return c.Next()
}
