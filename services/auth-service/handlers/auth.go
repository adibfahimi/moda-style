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

type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required,min=2"`
}

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

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

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

// GetProfile returns the current user's profile data
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

type updateProfileRequest struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=2"`
	Email *string `json:"email,omitempty" validate:"omitempty,email"`
}

// UpdateProfile updates the user's name, email, or preferences
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

type resetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

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

// generateResetToken creates a secure random token for password reset
func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
