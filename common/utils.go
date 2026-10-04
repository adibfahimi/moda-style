package common

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// validate is the shared validator instance. go-playground/validator caches
// struct metadata internally and is safe for concurrent use, so a single
// instance is reused for the lifetime of the process.
var validate = validator.New()

// ValidateStruct validates a value against its `validate:"..."` struct tags and
// returns the first validation error encountered, or nil when the value is
// valid.
func ValidateStruct(data interface{}) error {
	return validate.Struct(data)
}

// FormatValidationErrors converts a validator error into a single, human
// readable sentence (fields joined with ", ") suitable for returning to a
// client.
//
// When err is not a validator.ValidationErrors it falls back to a generic
// "validation failed" message so callers never leak internal error text.
func FormatValidationErrors(err error) string {
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrs {
			field := strings.ToLower(e.Field())
			switch e.Tag() {
			case "required":
				messages = append(messages, fmt.Sprintf("%s is required", field))
			case "email":
				messages = append(messages, fmt.Sprintf("%s must be a valid email", field))
			case "min":
				messages = append(messages, fmt.Sprintf("%s must be at least %s characters", field, e.Param()))
			case "max":
				messages = append(messages, fmt.Sprintf("%s must be at most %s characters", field, e.Param()))
			default:
				messages = append(messages, fmt.Sprintf("%s is invalid", field))
			}
		}
		return strings.Join(messages, ", ")
	}
	return "validation failed"
}

// SendErrorResponse writes a JSON error envelope ({"error": message}) using the
// given HTTP status code. Every service uses this so clients can rely on a
// single error shape.
func SendErrorResponse(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"error": message,
	})
}

// SendSuccessResponse writes a JSON body using the given HTTP status code.
func SendSuccessResponse(c *fiber.Ctx, status int, data fiber.Map) error {
	return c.Status(status).JSON(data)
}

// ParseAndValidate decodes the JSON request body into req and validates it
// against its `validate:"..."` struct tags.
//
// It reports success by returning true. When decoding or validation fails it
// writes a 400 error response itself (via SendErrorResponse) and returns false,
// so handlers can simply `if !ParseAndValidate(c, &body) { return nil }`.
func ParseAndValidate(c *fiber.Ctx, req interface{}) bool {
	// Parse request body
	if err := c.BodyParser(req); err != nil {
		SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
		return false
	}

	// Validate using validator tags
	if err := ValidateStruct(req); err != nil {
		SendErrorResponse(c, fiber.StatusBadRequest, FormatValidationErrors(err))
		return false
	}

	return true
}
