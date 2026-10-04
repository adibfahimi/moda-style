package common

import (
	"github.com/gofiber/fiber/v2"
)

// RequireAdmin is Fiber middleware that restricts a route to authenticated
// users whose token role is "admin".
//
// It must be registered after RequireAuth, which is what populates the "userID"
// and "role" values in the request Locals. It replies with 401 when the request
// was never authenticated and 403 when the caller is authenticated but is not
// an admin.
func RequireAdmin(c *fiber.Ctx) error {
	// RequireAuth stores the caller's id and role in the request Locals.
	if _, ok := c.Locals("userID").(uint); !ok {
		return SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	role, ok := c.Locals("role").(string)
	if !ok || role != "admin" {
		return SendErrorResponse(c, fiber.StatusForbidden, "Admin access required")
	}

	return c.Next()
}
