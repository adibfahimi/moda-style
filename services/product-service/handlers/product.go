package handlers

import (
	"strconv"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/product-service/database"
	"github.com/adibfahimi/moda-style/services/product-service/models"
	"github.com/gofiber/fiber/v2"
)

func getDescendantCategoryIDs(rootID uint) ([]uint, error) {
	ids := []uint{rootID}
	queue := []uint{rootID}

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		var children []models.Category
		if err := database.DB.
			Select("id").
			Where("parent_id = ? AND deleted_at IS NULL", currentID).
			Find(&children).Error; err != nil {
			return nil, err
		}

		for _, child := range children {
			ids = append(ids, child.ID)
			queue = append(queue, child.ID)
		}
	}

	return ids, nil
}

// ListProducts returns products with optional filters
// Query params: category, size, color, min_price, max_price, search, page, limit
func ListProducts(c *fiber.Ctx) error {
	query := database.DB.Model(&models.Product{}).Preload("Category").Preload("Sizes")

	// Filter by search query (name or description)
	if search := c.Query("search"); search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("products.name ILIKE ? OR products.description ILIKE ?", searchPattern, searchPattern)
	}

	// Filter by category
	if category := c.Query("category"); category != "" {
		categoryID, err := strconv.ParseUint(category, 10, 32)
		if err == nil {
			categoryIDs, err := getDescendantCategoryIDs(uint(categoryID))
			if err != nil {
				return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch category hierarchy")
			}
			query = query.Where("products.category_id IN ?", categoryIDs)
		} else {
			query = query.Joins("JOIN categories ON categories.id = products.category_id").
				Where("categories.name = ?", category)
		}
	}

	// Filter by size
	if size := c.Query("size"); size != "" {
		query = query.Joins("JOIN sizes ON sizes.product_id = products.id").
			Where("sizes.size = ? AND sizes.stock > 0", size)
	}

	// Filter by color
	if color := c.Query("color"); color != "" {
		query = query.Joins("JOIN sizes ON sizes.product_id = products.id").
			Where("sizes.color = ? AND sizes.stock > 0", color)
	}

	// Filter by price range
	if minPrice := c.Query("min_price"); minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			query = query.Where("price >= ?", price)
		}
	}
	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			query = query.Where("price <= ?", price)
		}
	}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Get total count
	var total int64
	query.Count(&total)

	// Get products
	var products []models.Product
	if err := query.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch products")
	}

	// Calculate stock for each product
	for i := range products {
		products[i].CalculateStock()
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"products": products,
		"page":     page,
		"limit":    limit,
		"total":    total,
	})
}

// GetProduct returns a specific product with sizes and stock information
func GetProduct(c *fiber.Ctx) error {
	id := c.Params("id")

	var product models.Product
	if err := database.DB.Preload("Category").Preload("Sizes").First(&product, id).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Product not found")
	}

	// Calculate total stock
	product.CalculateStock()

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"product": product,
	})
}
