// Package handlers implements the cart and wishlist HTTP surface.
//
// Every route is mounted behind common.RequireAuth, so handlers read the caller
// from c.Locals("userID") and return 401 when it is missing. Cart items are
// stored in this service's tables, while product and size details are read from
// the products/sizes tables that live in the shared database.
package handlers

import (
	"strconv"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/cart-service/database"
	"github.com/adibfahimi/moda-style/services/cart-service/models"
	"github.com/gofiber/fiber/v2"
)

// AddToCartRequest is the payload accepted by AddToCart.
type AddToCartRequest struct {
	ProductID uint `json:"product_id" validate:"required"`
	SizeID    uint `json:"size_id" validate:"required"`
	Quantity  int  `json:"quantity" validate:"required,min=1"`
}

// UpdateCartItemRequest is the payload accepted by UpdateCartItem.
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,min=1"`
}

// GetCart returns the authenticated user's cart with product details, the
// subtotal and the number of distinct lines.
//
// Product name, image and price plus the selected size, colour and remaining
// stock are resolved per item, so the cart can be rendered without extra calls.
// Items whose product or size row has disappeared keep zero values rather than
// failing the whole request.
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

// AddToCart adds a size variant to the cart, or increases the quantity when the
// same product/size pair is already present.
//
// It validates the payload, resolves the size to check that it exists, belongs
// to the requested product and has enough stock, then either merges the
// quantity into the existing line or creates a new one. Responses are 201 for a
// new line and 200 when an existing line was updated.
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

// UpdateCartItem sets the absolute quantity of one of the caller's cart lines.
//
// The item must belong to the authenticated user, and the requested quantity
// must not exceed the stock of its size variant. Unknown or foreign IDs return
// 404 so a user cannot probe other carts.
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

// RemoveFromCart deletes one of the caller's cart lines, returning 404 when the
// ID does not exist or belongs to another user.
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

// ClearCart removes every cart line of the authenticated user. It is
// idempotent: clearing an already empty cart still returns 200.
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

// GetWishlist returns the authenticated user's wishlist with product details and
// a computed InStock flag driven by the summed stock of all size variants.
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

// ToggleWishlistItem adds the product to the caller's wishlist when absent and
// removes it when present.
//
// The response always carries a "wishlisted" boolean so the client can update
// its button state: 201 for an add, 200 for a removal. Unknown product IDs
// return 404 before any write happens.
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
			"message":    "Removed from wishlist",
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
		"message":    "Added to wishlist",
		"wishlisted": true,
		"item":       wishlistItem,
	})
}
