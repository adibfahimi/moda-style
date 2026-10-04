// Package handlers implements the public HTTP surface of the product service:
// the product catalogue, its category tree and customer reviews.
//
// Read endpoints are public, while CreateReview sits behind
// common.RequireAuth and reads the caller identity from the request Locals.
// Errors and successes are written with the shared common response envelopes.
package handlers

import (
	"strconv"
	"strings"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/product-service/database"
	"github.com/adibfahimi/moda-style/services/product-service/models"
	"github.com/gofiber/fiber/v2"
)

// getDescendantCategoryIDs returns rootID followed by every category reachable
// through the ParentID chain, using an iterative breadth-first traversal.
//
// The result is used to make category filtering include subcategories, so
// requesting the "Women" category also returns products filed under "Dresses".
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

// ListProducts returns the product catalogue, optionally filtered and paginated.
//
// Supported query parameters:
//
//	search     – matches the product name or description, case-insensitively
//	category   – numeric category ID (includes all descendants) or a category name
//	size       – only products with that size in stock
//	color      – only products with that colour in stock
//	min_price  – inclusive lower bound on the product price
//	max_price  – inclusive upper bound on the product price
//	page       – 1-based page number (default 1)
//	limit      – page size, 1-100 (default 20)
//
// The response also carries the unpaginated total so clients can render
// pagination controls. Every returned product has its Stock field computed from
// its sizes.
func ListProducts(c *fiber.Ctx) error {
	query := database.DB.Model(&models.Product{}).Preload("Category").Preload("Sizes")

	// Filter by search query (name or description). LOWER()/LIKE is used instead
	// of ILIKE so the query works on both PostgreSQL and SQLite.
	if search := c.Query("search"); search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(products.name) LIKE ? OR LOWER(products.description) LIKE ?", searchPattern, searchPattern)
	}

	// Filter by category
	if category := c.Query("category"); category != "" {
		// A numeric value is a category ID: products are matched against the whole
		// subtree. Anything else is treated as a category name.
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

// GetProduct returns a single product with its category and sizes preloaded.
//
// The Stock field is computed from the product's sizes, and an unknown or
// soft-deleted ID yields 404 Not Found.
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
