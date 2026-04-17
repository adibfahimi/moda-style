package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/admin-service/database"
	"github.com/gofiber/fiber/v2"
)

// Product represents the product model for admin operations
type Product struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name" validate:"required,min=2"`
	Description string  `json:"description"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	CategoryID  uint    `json:"category_id" validate:"required"`
	ImageURL    string  `json:"image_url"`
}

// Size represents product size/variant
type Size struct {
	ID        uint   `json:"id"`
	ProductID uint   `json:"product_id"`
	Size      string `json:"size" validate:"required"`
	Color     string `json:"color" validate:"required"`
	Stock     int    `json:"stock" validate:"gte=0"`
}

const maxProductImageSize = 5 * 1024 * 1024 // 5MB

type ProductImage struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	UpdatedAt string `json:"updated_at"`
}

// ListProductImages returns previously uploaded product images.
func ListProductImages(c *fiber.Ctx) error {
	uploadDir := filepath.Join("uploads", "products")
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to access uploads directory")
	}

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to list uploaded images")
	}

	images := make([]ProductImage, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}

		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}

		path := "/api/v1/admin/uploads/products/" + entry.Name()
		images = append(images, ProductImage{
			Name:      entry.Name(),
			URL:       c.BaseURL() + path,
			Size:      fileInfo.Size(),
			UpdatedAt: fileInfo.ModTime().UTC().Format(time.RFC3339),
		})
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].UpdatedAt > images[j].UpdatedAt
	})

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"images": images,
		"total":  len(images),
	})
}

// UploadProductImage uploads a product image and returns a public URL.
func UploadProductImage(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Image file is required")
	}

	if fileHeader.Size <= 0 {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Uploaded file is empty")
	}

	if fileHeader.Size > maxProductImageSize {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Image size exceeds 5MB limit")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExt := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}

	if !allowedExt[ext] {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Only JPG, PNG, and WEBP files are allowed")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Failed to read uploaded image")
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	detectedContentType := strings.ToLower(http.DetectContentType(buffer[:n]))

	contentType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	if idx := strings.Index(contentType, ";"); idx > -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectedContentType
	}

	allowedContentTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}

	if !allowedContentTypes[contentType] {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid image file type. Please upload JPG, PNG, or WEBP")
	}

	uploadDir := filepath.Join("uploads", "products")
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to prepare upload directory")
	}

	fileName := fmt.Sprintf("product-%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveFile(fileHeader, savePath); err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to save uploaded image")
	}

	imagePath := "/api/v1/admin/uploads/products/" + fileName

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message":    "Image uploaded successfully",
		"image_url":  c.BaseURL() + imagePath,
		"image_path": imagePath,
	})
}

// CreateProduct creates a new product
func CreateProduct(c *fiber.Ctx) error {
	var req Product
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	// Verify category exists
	var categoryExists bool
	database.DB.Table("categories").
		Select("count(*) > 0").
		Where("id = ? AND deleted_at IS NULL", req.CategoryID).
		Scan(&categoryExists)

	if !categoryExists {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid category ID")
	}

	// Create product
	if err := database.DB.Table("products").Create(&req).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to create product")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "created", "product", req.ID, "Created product: "+req.Name, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusCreated, fiber.Map{
		"message": "Product created successfully",
		"product": req,
	})
}

// UpdateProduct updates an existing product
func UpdateProduct(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid product ID")
	}

	var req Product
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	// Check if product exists
	var exists bool
	database.DB.Table("products").
		Select("count(*) > 0").
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(&exists)

	if !exists {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Product not found")
	}

	// Verify category exists if provided
	if req.CategoryID > 0 {
		var categoryExists bool
		database.DB.Table("categories").
			Select("count(*) > 0").
			Where("id = ? AND deleted_at IS NULL", req.CategoryID).
			Scan(&categoryExists)

		if !categoryExists {
			return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid category ID")
		}
	}

	// Update product
	if err := database.DB.Table("products").Where("id = ?", id).Updates(&req).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to update product")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "updated", "product", uint(id), "Updated product: "+req.Name, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Product updated successfully",
		"product": req,
	})
}

// DeleteProduct soft deletes a product
func DeleteProduct(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid product ID")
	}

	// Check if product exists
	var productName string
	if err := database.DB.Table("products").
		Select("name").
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(&productName).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Product not found")
	}

	// Soft delete product
	if err := database.DB.Table("products").Where("id = ?", id).Update("deleted_at", "NOW()").Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete product")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "deleted", "product", uint(id), "Deleted product: "+productName, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Product deleted successfully",
	})
}

// GetProductStats returns detailed product statistics
func GetProductStats(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	type ProductStat struct {
		ID            uint    `json:"id"`
		Name          string  `json:"name"`
		Description   string  `json:"description"`
		CategoryID    uint    `json:"category_id"`
		CategoryName  string  `json:"category_name"`
		Price         float64 `json:"price"`
		Stock         int     `json:"stock"`
		ImageURL      string  `json:"image_url"`
		ReviewCount   int64   `json:"review_count"`
		AverageRating float64 `json:"average_rating"`
		WishlistCount int64   `json:"wishlist_count"`
	}

	var products []ProductStat
	var total int64

	// Count total products
	database.DB.Table("products").Where("deleted_at IS NULL").Count(&total)

	// Get product stats with joins
	database.DB.Table("products").
		Select(`
			products.id,
			products.name,
			products.description,
			products.category_id,
			categories.name as category_name,
			products.price,
			products.image_url,
			COALESCE(SUM(sizes.stock), 0) as stock,
			COUNT(DISTINCT reviews.id) as review_count,
			COALESCE(AVG(reviews.rating), 0) as average_rating,
			COUNT(DISTINCT wishlist_items.id) as wishlist_count
		`).
		Joins("LEFT JOIN categories ON products.category_id = categories.id").
		Joins("LEFT JOIN sizes ON products.id = sizes.product_id AND sizes.deleted_at IS NULL").
		Joins("LEFT JOIN reviews ON products.id = reviews.product_id AND reviews.deleted_at IS NULL").
		Joins("LEFT JOIN wishlist_items ON products.id = wishlist_items.product_id AND wishlist_items.deleted_at IS NULL").
		Where("products.deleted_at IS NULL").
		Group("products.id, categories.name").
		Order("products.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&products)

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"products": products,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetProductSizes returns all size variants for a product.
func GetProductSizes(c *fiber.Ctx) error {
	productID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid product ID")
	}

	// Check if product exists
	var exists bool
	database.DB.Table("products").
		Select("count(*) > 0").
		Where("id = ? AND deleted_at IS NULL", productID).
		Scan(&exists)

	if !exists {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Product not found")
	}

	var sizes []Size
	if err := database.DB.Table("sizes").
		Where("product_id = ? AND deleted_at IS NULL", productID).
		Order("id ASC").
		Find(&sizes).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch product sizes")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"sizes": sizes,
		"total": len(sizes),
	})
}

// AddProductSize adds a new size variant to a product
func AddProductSize(c *fiber.Ctx) error {
	productID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid product ID")
	}

	var req Size
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	// Check if product exists
	var exists bool
	database.DB.Table("products").
		Select("count(*) > 0").
		Where("id = ? AND deleted_at IS NULL", productID).
		Scan(&exists)

	if !exists {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Product not found")
	}

	req.ProductID = uint(productID)

	// Create size
	if err := database.DB.Table("sizes").Create(&req).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to add product size")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "created", "size", req.ID, "Added size variant to product", c.IP())

	return common.SendSuccessResponse(c, fiber.StatusCreated, fiber.Map{
		"message": "Size added successfully",
		"size":    req,
	})
}

// UpdateProductSize updates a product size variant
func UpdateProductSize(c *fiber.Ctx) error {
	sizeID, err := strconv.ParseUint(c.Params("sizeId"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid size ID")
	}

	var req Size
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	// Check if size exists
	var exists bool
	database.DB.Table("sizes").
		Select("count(*) > 0").
		Where("id = ? AND deleted_at IS NULL", sizeID).
		Scan(&exists)

	if !exists {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Size not found")
	}

	// Update size
	if err := database.DB.Table("sizes").Where("id = ?", sizeID).Updates(&req).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to update size")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "updated", "size", uint(sizeID), "Updated product size variant", c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Size updated successfully",
		"size":    req,
	})
}

// DeleteProductSize removes a size variant
func DeleteProductSize(c *fiber.Ctx) error {
	sizeID, err := strconv.ParseUint(c.Params("sizeId"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid size ID")
	}

	// Check if size exists
	var exists bool
	database.DB.Table("sizes").
		Select("count(*) > 0").
		Where("id = ? AND deleted_at IS NULL", sizeID).
		Scan(&exists)

	if !exists {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Size not found")
	}

	// Delete size
	if err := database.DB.Table("sizes").Where("id = ?", sizeID).Update("deleted_at", "NOW()").Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete size")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "deleted", "size", uint(sizeID), "Deleted product size variant", c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Size deleted successfully",
	})
}
