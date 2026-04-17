package common

import (
	"github.com/gofiber/fiber/v2"
)

// RequireAdmin middleware checks if the user has admin role
// This middleware should be used after RequireAuth middleware
func RequireAdmin(c *fiber.Ctx) error {
	// Get user role from token claims
	// Note: We need to add role to the JWT claims
	_, ok := c.Locals("userID").(uint)
	if !ok {
		return SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	// For now, we'll check the role from locals
	// The JWT generation should be updated to include role
	role, ok := c.Locals("role").(string)
	if !ok || role != "admin" {
		return SendErrorResponse(c, fiber.StatusForbidden, "Admin access required")
	}

	return c.Next()
}
