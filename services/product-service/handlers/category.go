package handlers

import (
	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/product-service/database"
	"github.com/adibfahimi/moda-style/services/product-service/models"
	"github.com/gofiber/fiber/v2"
)

// ListCategories returns all clothing categories
func ListCategories(c *fiber.Ctx) error {
	var categories []models.Category
	if err := database.DB.
		Preload("Parent").
		Order("parent_id ASC NULLS FIRST").
		Order("name ASC").
		Find(&categories).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch categories")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"categories": categories,
	})
}
