package handlers

import (
	"strconv"
	"strings"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/admin-service/database"
	"github.com/gofiber/fiber/v2"
)

// Category represents the category model for admin operations
type Category struct {
	ID       uint   `json:"id"`
	Name     string `json:"name" validate:"required,min=2"`
	Slug     string `json:"slug" validate:"omitempty,min=2"`
	ParentID *uint  `json:"parent_id,omitempty"`
}

// GetCategories returns all categories with product counts
func GetCategories(c *fiber.Ctx) error {
	type CategoryWithCount struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		ParentID     *uint  `json:"parent_id,omitempty"`
		ParentName   string `json:"parent_name,omitempty"`
		ProductCount int64  `json:"product_count"`
		CreatedAt    string `json:"created_at"`
	}

	var categories []CategoryWithCount

	database.DB.Table("categories").
		Select(`
			categories.id,
			categories.name,
			categories.slug,
			categories.parent_id,
			parent_categories.name as parent_name,
			COUNT(DISTINCT products.id) as product_count,
			categories.created_at
		`).
		Joins("LEFT JOIN categories AS parent_categories ON categories.parent_id = parent_categories.id").
		Joins("LEFT JOIN products ON categories.id = products.category_id AND products.deleted_at IS NULL").
		Where("categories.deleted_at IS NULL").
		Group("categories.id, parent_categories.name").
		Order("categories.name ASC").
		Scan(&categories)

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"categories": categories,
		"total":      len(categories),
	})
}

// CreateCategory creates a new category
func CreateCategory(c *fiber.Ctx) error {
	var req Category
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	// Generate slug from name if not provided
	if req.Slug == "" {
		req.Slug = generateSlug(req.Name)
	}

	if req.ParentID != nil {
		var parentExists bool
		database.DB.Table("categories").
			Select("count(*) > 0").
			Where("id = ? AND deleted_at IS NULL", *req.ParentID).
			Scan(&parentExists)

		if !parentExists {
			return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid parent category")
		}
	}

	// Check if category with same name or slug exists under same parent
	var exists bool
	query := database.DB.Table("categories").
		Select("count(*) > 0").
		Where("(name = ? OR slug = ?) AND deleted_at IS NULL", req.Name, req.Slug)

	if req.ParentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *req.ParentID)
	}

	query.Scan(&exists)

	if exists {
		return common.SendErrorResponse(c, fiber.StatusConflict, "Category with this name or slug already exists under the selected parent")
	}

	// Create category
	if err := database.DB.Table("categories").Create(&req).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to create category")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "created", "category", req.ID, "Created category: "+req.Name, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusCreated, fiber.Map{
		"message":  "Category created successfully",
		"category": req,
	})
}

// UpdateCategory updates an existing category
func UpdateCategory(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid category ID")
	}

	var req Category
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	// Check if category exists
	var exists bool
	database.DB.Table("categories").
		Select("count(*) > 0").
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(&exists)

	if !exists {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Category not found")
	}

	if req.Slug == "" {
		req.Slug = generateSlug(req.Name)
	}

	if req.ParentID != nil {
		if *req.ParentID == uint(id) {
			return common.SendErrorResponse(c, fiber.StatusBadRequest, "Category cannot be its own parent")
		}

		var parentExists bool
		database.DB.Table("categories").
			Select("count(*) > 0").
			Where("id = ? AND deleted_at IS NULL", *req.ParentID).
			Scan(&parentExists)

		if !parentExists {
			return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid parent category")
		}

		// Prevent cycles by walking up the parent chain.
		currentParentID := req.ParentID
		for currentParentID != nil {
			if *currentParentID == uint(id) {
				return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid parent category hierarchy")
			}

			var nextParentID *uint
			database.DB.Table("categories").
				Select("parent_id").
				Where("id = ? AND deleted_at IS NULL", *currentParentID).
				Scan(&nextParentID)
			currentParentID = nextParentID
		}
	}

	// Check if name or slug conflicts with another category
	var conflict bool
	conflictQuery := database.DB.Table("categories").
		Select("count(*) > 0").
		Where("(name = ? OR slug = ?) AND id != ? AND deleted_at IS NULL", req.Name, req.Slug, id)

	if req.ParentID == nil {
		conflictQuery = conflictQuery.Where("parent_id IS NULL")
	} else {
		conflictQuery = conflictQuery.Where("parent_id = ?", *req.ParentID)
	}

	conflictQuery.Scan(&conflict)

	if conflict {
		return common.SendErrorResponse(c, fiber.StatusConflict, "Category with this name or slug already exists under the selected parent")
	}

	// Update category
	if err := database.DB.Table("categories").Where("id = ?", id).Updates(&req).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to update category")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "updated", "category", uint(id), "Updated category: "+req.Name, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message":  "Category updated successfully",
		"category": req,
	})
}

// DeleteCategory soft deletes a category
func DeleteCategory(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid category ID")
	}

	// Check if category exists
	var categoryName string
	if err := database.DB.Table("categories").
		Select("name").
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(&categoryName).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Category not found")
	}

	// Check if category has products
	var productCount int64
	database.DB.Table("products").
		Where("category_id = ? AND deleted_at IS NULL", id).
		Count(&productCount)

	if productCount > 0 {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Cannot delete category with existing products")
	}

	// Check if category has children
	var childCount int64
	database.DB.Table("categories").
		Where("parent_id = ? AND deleted_at IS NULL", id).
		Count(&childCount)

	if childCount > 0 {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Cannot delete category with child categories")
	}

	// Soft delete category
	if err := database.DB.Table("categories").Where("id = ?", id).Update("deleted_at", "NOW()").Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete category")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "deleted", "category", uint(id), "Deleted category: "+categoryName, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Category deleted successfully",
	})
}

// generateSlug creates a URL-friendly slug from a string
func generateSlug(s string) string {
	// Convert to lowercase
	slug := strings.ToLower(s)

	// Replace spaces and special characters with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "&", "and")

	// Remove any characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, char := range slug {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			result.WriteRune(char)
		}
	}

	// Remove consecutive hyphens
	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}
