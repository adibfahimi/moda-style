package handlers

import (
	"strconv"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/cart-service/database"
	"github.com/adibfahimi/moda-style/services/cart-service/models"
	"github.com/gofiber/fiber/v2"
)

type AddToCartRequest struct {
	ProductID uint `json:"product_id" validate:"required"`
	SizeID    uint `json:"size_id" validate:"required"`
	Quantity  int  `json:"quantity" validate:"required,min=1"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,min=1"`
}

func GetCart(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	var cartItems []models.CartItem
	if err := database.DB.Where("user_id = ?", userID).Find(&cartItems).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch cart")
	}

	// Get product details for each cart item
	var items []models.CartItemWithProduct
	for _, item := range cartItems {
		var productInfo struct {
			Name     string
			ImageURL string
			Price    float64
		}
		var sizeInfo struct {
			Size  string
			Color string
			Stock int
		}

		// Fetch product info from product-service database (shared DB)
		database.DB.Table("products").Select("name, image_url, price").Where("id = ?", item.ProductID).Scan(&productInfo)
		database.DB.Table("sizes").Select("size, color, stock").Where("id = ?", item.SizeID).Scan(&sizeInfo)

		items = append(items, models.CartItemWithProduct{
			ID:        item.ID,
			ProductID: item.ProductID,
			Name:      productInfo.Name,
			ImageURL:  productInfo.ImageURL,
			Price:     productInfo.Price,
			Size:      sizeInfo.Size,
			Color:     sizeInfo.Color,
			SizeID:    item.SizeID,
			Quantity:  item.Quantity,
			Stock:     sizeInfo.Stock,
		})
	}

	// Calculate totals
	var subtotal float64
	for _, item := range items {
		subtotal += item.Price * float64(item.Quantity)
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"items":    items,
		"subtotal": subtotal,
		"count":    len(items),
	})
}

func AddToCart(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	var req AddToCartRequest
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	// Check if product and size exist and have stock
	var sizeInfo struct {
		ProductID uint
		Stock     int
	}
	if err := database.DB.Table("sizes").Select("product_id, stock").Where("id = ?", req.SizeID).Scan(&sizeInfo).Error; err != nil || sizeInfo.ProductID == 0 {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Size not found")
	}

	if sizeInfo.ProductID != req.ProductID {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Size does not belong to this product")
	}

	if sizeInfo.Stock < req.Quantity {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Not enough stock available")
	}

	// Check if item already exists in cart
	var existingItem models.CartItem
	result := database.DB.Where("user_id = ? AND product_id = ? AND size_id = ?", userID, req.ProductID, req.SizeID).First(&existingItem)

	if result.RowsAffected > 0 {
		// Update quantity
		newQuantity := existingItem.Quantity + req.Quantity
		if newQuantity > sizeInfo.Stock {
			return common.SendErrorResponse(c, fiber.StatusBadRequest, "Not enough stock available")
		}
		existingItem.Quantity = newQuantity
		if err := database.DB.Save(&existingItem).Error; err != nil {
			return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to update cart")
		}
		return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
			"message": "Cart updated",
			"item":    existingItem,
		})
	}

	// Create new cart item
	cartItem := models.CartItem{
		UserID:    userID,
		ProductID: req.ProductID,
		SizeID:    req.SizeID,
		Quantity:  req.Quantity,
	}

	if err := database.DB.Create(&cartItem).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to add to cart")
	}

	return common.SendSuccessResponse(c, fiber.StatusCreated, fiber.Map{
		"message": "Item added to cart",
		"item":    cartItem,
	})
}

func UpdateCartItem(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid cart item ID")
	}

	var req UpdateCartItemRequest
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	var cartItem models.CartItem
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&cartItem).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Cart item not found")
	}

	// Check stock availability
	var stock int
	database.DB.Table("sizes").Select("stock").Where("id = ?", cartItem.SizeID).Scan(&stock)
	if req.Quantity > stock {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Not enough stock available")
	}

	cartItem.Quantity = req.Quantity
	if err := database.DB.Save(&cartItem).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to update cart item")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Cart item updated",
		"item":    cartItem,
	})
}

func RemoveFromCart(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid cart item ID")
	}

	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CartItem{})
	if result.RowsAffected == 0 {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Cart item not found")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Item removed from cart",
	})
}

func ClearCart(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	if err := database.DB.Where("user_id = ?", userID).Delete(&models.CartItem{}).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to clear cart")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Cart cleared",
	})
}

func GetWishlist(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	var wishlistItems []models.WishlistItem
	if err := database.DB.Where("user_id = ?", userID).Find(&wishlistItems).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch wishlist")
	}

	// Get product details for each wishlist item
	var items []models.WishlistItemWithProduct
	for _, item := range wishlistItems {
		var productInfo struct {
			Name     string
			ImageURL string
			Price    float64
		}
		database.DB.Table("products").Select("name, image_url, price").Where("id = ?", item.ProductID).Scan(&productInfo)

		// Check if any size has stock
		var totalStock int64
		database.DB.Table("sizes").Where("product_id = ?", item.ProductID).Select("SUM(stock)").Scan(&totalStock)

		items = append(items, models.WishlistItemWithProduct{
			ID:        item.ID,
			ProductID: item.ProductID,
			Name:      productInfo.Name,
			ImageURL:  productInfo.ImageURL,
			Price:     productInfo.Price,
			InStock:   totalStock > 0,
		})
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"items": items,
		"count": len(items),
	})
}

func ToggleWishlistItem(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return common.SendErrorResponse(c, fiber.StatusUnauthorized, "User not authenticated")
	}

	productID, err := strconv.ParseUint(c.Params("product_id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid product ID")
	}

	// Check if product exists
	var productExists int64
	database.DB.Table("products").Where("id = ?", productID).Count(&productExists)
	if productExists == 0 {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Product not found")
	}

	// Check if item already exists in wishlist
	var existingItem models.WishlistItem
	result := database.DB.Where("user_id = ? AND product_id = ?", userID, productID).First(&existingItem)

	if result.RowsAffected > 0 {
		// Remove from wishlist
		if err := database.DB.Delete(&existingItem).Error; err != nil {
			return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to remove from wishlist")
		}
		return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
			"message":   "Removed from wishlist",
			"wishlisted": false,
		})
	}

	// Add to wishlist
	wishlistItem := models.WishlistItem{
		UserID:    userID,
		ProductID: uint(productID),
	}

	if err := database.DB.Create(&wishlistItem).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to add to wishlist")
	}

	return common.SendSuccessResponse(c, fiber.StatusCreated, fiber.Map{
		"message":   "Added to wishlist",
		"wishlisted": true,
		"item":      wishlistItem,
	})
}
