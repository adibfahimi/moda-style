package common

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// Validator instance
var validate = validator.New()

// ValidateStruct validates a struct using validator tags
func ValidateStruct(data interface{}) error {
	return validate.Struct(data)
}

// FormatValidationErrors formats validator errors into a readable string
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

// SendErrorResponse sends a standardized error response
func SendErrorResponse(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"error": message,
	})
}

// SendSuccessResponse sends a standardized success response
func SendSuccessResponse(c *fiber.Ctx, status int, data fiber.Map) error {
	return c.Status(status).JSON(data)
}

// ParseAndValidate parses request body and validates it using validator tags
// Returns true if successful, false if an error response was sent
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
