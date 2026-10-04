// Package handlers implements the HTTP surface of the auth service: account
// registration, login, profile read/update and password-reset requests.
//
// Every handler follows the same contract:
//
//   - request bodies are decoded and validated with common.ParseAndValidate,
//     which writes a 400 response on failure and reports it by returning false;
//   - responses use the shared common.SendSuccessResponse /
//     common.SendErrorResponse envelopes so all services fail identically;
//   - authenticated handlers read the caller identity from the request Locals
//     that common.RequireAuth populates ("userID", "email", "userName", "role").
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/auth-service/database"
	"github.com/adibfahimi/moda-style/services/auth-service/models"
	"github.com/gofiber/fiber/v2"
)

// registerRequest is the JSON body accepted by Register.
type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required,min=2"`
}

// Register creates a new user account and returns a signed access token.
//
// The password is stored as a bcrypt hash only; the plaintext value never
// reaches the database. An email that is already registered yields
// 409 Conflict, invalid input yields 400 Bad Request, and success yields
// 201 Created with the token plus the public fields of the created user.
func Register(c *fiber.Ctx) error {
	var req registerRequest

	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return common.SendErrorResponse(c, fiber.StatusConflict, "Email already registered")
	}

	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := user.HashPassword(); err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to create user")
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to create user")
	}

	token, err := common.GenerateToken(user.ID, user.Email, user.Name, user.Role)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	return common.SendSuccessResponse(c, fiber.StatusCreated, fiber.Map{
		"message": "Registration successful",
		"token":   token,
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

// loginRequest is the JSON body accepted by Login.
type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// Login authenticates an existing user and returns a signed access token.
//
// An unknown email and a wrong password both produce the same
// 401 Unauthorized "Invalid email or password" response so the endpoint cannot
// be used to enumerate registered accounts.
func Login(c *fiber.Ctx) error {
	var req loginRequest

	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "Invalid email or password")
	}

	if !user.CheckPassword(req.Password) {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "Invalid email or password")
	}

	token, err := common.GenerateToken(user.ID, user.Email, user.Name, user.Role)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Login successful",
		"token":   token,
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

// GetProfile returns the profile of the authenticated user.
//
// The identity is read from the request Locals populated by common.RequireAuth,
// so the route must be mounted behind that middleware. A missing identity
// yields 401 Unauthorized and an unknown user 404 Not Found.
func GetProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "User not found")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"user": fiber.Map{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	})
}

// updateProfileRequest is the JSON body accepted by UpdateProfile.
//
// Both fields are optional pointers so an omitted field keeps its current value
// while a present-but-invalid value is still rejected by the validator.
type updateProfileRequest struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=2"`
	Email *string `json:"email,omitempty" validate:"omitempty,email"`
}

// UpdateProfile changes the name and/or email of the authenticated user.
//
// Only the fields present in the request body are written. Changing the email
// to a value owned by another account yields 409 Conflict; the account's role
// and password cannot be modified through this endpoint.
func UpdateProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	var req updateProfileRequest
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "User not found")
	}

	if req.Name != nil {
		user.Name = *req.Name
	}

	if req.Email != nil {
		var existingUser models.User
		if err := database.DB.Where("email = ? AND id != ?", *req.Email, userID).First(&existingUser).Error; err == nil {
			return common.SendErrorResponse(c, fiber.StatusConflict, "Email already in use")
		}
		user.Email = *req.Email
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to update profile")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Profile updated successfully",
		"user": fiber.Map{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"updated_at": user.UpdatedAt,
		},
	})
}

// resetPasswordRequest is the JSON body accepted by ResetPassword.
type resetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPassword starts the password-reset flow for an email address.
//
// The response is the same 200 OK message whether or not the address belongs to
// a real account, so the endpoint cannot be used to discover which emails are
// registered. For a known account a random single-use token valid for one hour
// is stored on the user record.
//
// TODO: deliver the token to the user by email instead of logging it to stdout.
func ResetPassword(c *fiber.Ctx) error {
	var req resetPasswordRequest
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
			"message": "If the email exists, a password reset link has been sent",
		})
	}

	resetToken, err := generateResetToken()
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate reset token")
	}

	user.ResetToken = &resetToken
	expiryTime := time.Now().Add(1 * time.Hour)
	user.ResetTokenExpiry = &expiryTime

	if err := database.DB.Save(&user).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to process reset request")
	}

	// TODO: In production, send email with reset link containing the token
	fmt.Printf("Password reset token for %s: %s\n", user.Email, resetToken)

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// generateResetToken returns a cryptographically secure password-reset token:
// 32 random bytes from crypto/rand, hex encoded (64 characters).
func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
