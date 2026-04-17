package handlers

import (
	"strconv"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/product-service/database"
	"github.com/adibfahimi/moda-style/services/product-service/models"
	"github.com/gofiber/fiber/v2"
)

type CreateReviewRequest struct {
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"required,min=10,max=1000"`
}

// GetProductReviews returns customer feedback for a product
func GetProductReviews(c *fiber.Ctx) error {
	productID := c.Params("id")

	// Verify product exists
	var product models.Product
	if err := database.DB.First(&product, productID).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Product not found")
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

	// Get total count and average rating
	var total int64
	var avgRating float64
	database.DB.Model(&models.Review{}).Where("product_id = ?", productID).Count(&total)
	database.DB.Model(&models.Review{}).Where("product_id = ?", productID).Select("AVG(rating)").Scan(&avgRating)

	// Get reviews
	var reviews []models.Review
	if err := database.DB.Where("product_id = ?", productID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch reviews")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"reviews":        reviews,
		"total":          total,
		"average_rating": avgRating,
		"page":           page,
		"limit":          limit,
	})
}

// CreateReview submits a new review/rating for a product
func CreateReview(c *fiber.Ctx) error {
	productID := c.Params("id")

	// Get user ID and userName from auth middleware
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	// TODO: bro we dont have any userName
	userName, _ := c.Locals("userName").(string)
	if userName == "" {
		userName = "Anonymous" // Fallback
	}

	var req CreateReviewRequest
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	// Verify product exists
	var product models.Product
	if err := database.DB.First(&product, productID).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Product not found")
	}

	// Check if user already reviewed this product
	var existingReview models.Review
	if err := database.DB.Where("product_id = ? AND user_id = ?", productID, userID).First(&existingReview).Error; err == nil {
		return common.SendErrorResponse(c, fiber.StatusConflict, "You have already reviewed this product")
	}

	// Create review
	review := models.Review{
		ProductID: product.ID,
		UserID:    userID,
		UserName:  userName,
		Rating:    req.Rating,
		Comment:   req.Comment,
	}

	if err := database.DB.Create(&review).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to create review")
	}

	return common.SendSuccessResponse(c, fiber.StatusCreated, fiber.Map{
		"message": "Review submitted successfully",
		"review":  review,
	})
}
