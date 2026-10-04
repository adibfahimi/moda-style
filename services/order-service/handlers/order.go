// Package handlers implements the order lifecycle of the store: checkout from
// the cart, payment processing, order history and cancellation.
//
// Every route runs behind common.RequireAuth, so handlers read the caller from
// c.Locals("userID") and return 401 when it is missing. Orders are stored in the
// order tables while the cart and product/size data are read from the shared
// database, which is also where stock is decremented after a successful
// payment. Payments are simulated: no Stripe call is made yet.
package handlers

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/adibfahimi/moda-style/services/order-service/database"
	"github.com/adibfahimi/moda-style/services/order-service/models"
	"github.com/gofiber/fiber/v2"
)

// CartItem mirrors the cart lines of the cart service that this service reads
// directly from the shared database during checkout.
type CartItem struct {
	ID        uint    `json:"id"`
	ProductID uint    `json:"product_id"`
	SizeID    uint    `json:"size_id"`
	Quantity  int     `json:"quantity"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Size      string  `json:"size"`
	Color     string  `json:"color"`
}

// CreateOrderRequest represents the request body for creating an order
type CreateOrderRequest struct {
	ShippingAddress string `json:"shipping_address"`
	PaymentMethod   string `json:"payment_method"`
	Notes           string `json:"notes"`
}

// ProcessPaymentRequest represents the request body for processing payment
type ProcessPaymentRequest struct {
	PaymentIntentID string `json:"payment_intent_id"`
}

// CreateOrder turns the authenticated user's cart into an order.
//
// The shipping address is mandatory and the payment method defaults to "card".
// For every cart line the matching product and size are loaded from the shared
// database; a missing product or insufficient stock aborts the checkout with
// 400 before anything is written. The order is created with status and payment
// status "pending", a generated order number and one OrderItem per cart line,
// and returned with its items preloaded. The cart is only emptied later, once
// the payment succeeds.
func CreateOrder(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not authenticated"})
	}

	var req CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate required fields
	if req.ShippingAddress == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Shipping address is required"})
	}

	if req.PaymentMethod == "" {
		req.PaymentMethod = "card"
	}

	// Fetch cart items from cart-service database
	// Note: In production, you might want to call the cart service API instead
	var cartItems []struct {
		ID        uint
		ProductID uint
		SizeID    uint
		Quantity  int
	}

	if err := database.DB.Table("cart_items").
		Where("user_id = ?", userID).
		Find(&cartItems).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch cart"})
	}

	if len(cartItems) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Cart is empty"})
	}

	// Fetch product details for cart items
	type ProductDetail struct {
		ProductID uint
		Name      string
		Price     float64
		Size      string
		Color     string
		Stock     int
	}

	var productDetails []ProductDetail
	if err := database.DB.Raw(`
		SELECT 
			p.id as product_id,
			p.name,
			p.price,
			s.size,
			s.color,
			s.stock
		FROM products p
		JOIN sizes s ON p.id = s.product_id
		WHERE s.id IN (?)
	`, func() []uint {
		sizeIDs := make([]uint, len(cartItems))
		for i, item := range cartItems {
			sizeIDs[i] = item.SizeID
		}
		return sizeIDs
	}()).Scan(&productDetails).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch product details"})
	}

	// Create product detail map for easy lookup
	productMap := make(map[uint]ProductDetail)
	for _, pd := range productDetails {
		productMap[pd.ProductID] = pd
	}

	// Calculate total and create order items
	var totalAmount float64
	orderItems := make([]models.OrderItem, 0, len(cartItems))

	for _, cartItem := range cartItems {
		product, exists := productMap[cartItem.ProductID]
		if !exists {
			return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("Product %d not found", cartItem.ProductID)})
		}

		// Check stock
		if product.Stock < cartItem.Quantity {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Insufficient stock for %s. Available: %d, Requested: %d",
					product.Name, product.Stock, cartItem.Quantity),
			})
		}

		subtotal := product.Price * float64(cartItem.Quantity)
		totalAmount += subtotal

		sizeID := cartItem.SizeID
		orderItems = append(orderItems, models.OrderItem{
			ProductID:   cartItem.ProductID,
			ProductName: product.Name,
			SizeID:      &sizeID,
			Size:        product.Size,
			Color:       product.Color,
			Quantity:    cartItem.Quantity,
			Price:       product.Price,
			Subtotal:    subtotal,
		})
	}

	// Generate unique order number
	orderNumber := generateOrderNumber()

	// Create order
	order := models.Order{
		UserID:          userID,
		OrderNumber:     orderNumber,
		Status:          "pending",
		TotalAmount:     totalAmount,
		ShippingAddress: req.ShippingAddress,
		PaymentMethod:   req.PaymentMethod,
		PaymentStatus:   "pending",
		Notes:           req.Notes,
		Items:           orderItems,
	}

	if err := database.DB.Create(&order).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create order"})
	}

	// Reload order with items
	if err := database.DB.Preload("Items").First(&order, order.ID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load order details"})
	}

	return c.Status(201).JSON(fiber.Map{
		"order":   order,
		"message": "Order created successfully",
	})
}

// ProcessPayment settles an order with a (simulated) payment intent.
//
// The order must belong to the caller and must not be paid yet. A simulated
// transaction row is written for both outcomes: on success the order moves to
// "processing"/"paid", the stock of every ordered size is decremented and the
// user's cart is emptied; on decline the order is marked "failed" and returned
// with 400. No real Stripe request is made yet.
func ProcessPayment(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not authenticated"})
	}
	orderID := c.Params("id")

	var req ProcessPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.PaymentIntentID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Payment intent ID is required"})
	}

	// Find order
	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}

	// Check if already paid
	if order.PaymentStatus == "paid" {
		return c.Status(400).JSON(fiber.Map{"error": "Order already paid"})
	}

	// FAKE PAYMENT PROCESSING - Replace this with real Stripe integration
	// In production, you would:
	// 1. Verify payment intent with Stripe API
	// 2. Check payment status
	// 3. Handle webhooks for payment confirmations

	paymentSuccess := simulatePaymentProcessing(req.PaymentIntentID)

	// Create payment transaction record
	transaction := models.PaymentTransaction{
		OrderID:         order.ID,
		PaymentIntentID: req.PaymentIntentID,
		Amount:          order.TotalAmount,
		Currency:        "usd",
		Status:          "pending",
		PaymentMethod:   order.PaymentMethod,
	}

	if paymentSuccess {
		// Payment succeeded
		transaction.Status = "succeeded"
		order.PaymentStatus = "paid"
		order.PaymentIntentID = req.PaymentIntentID
		order.Status = "processing" // Move to processing once paid

		// Update stock quantities
		if err := decreaseProductStock(&order); err != nil {
			transaction.ErrorMessage = "Payment succeeded but failed to update stock: " + err.Error()
		} else {
			// Clear user's cart
			if err := clearUserCart(userID); err != nil {
				// Log error but don't fail the payment
				fmt.Printf("Warning: Failed to clear cart for user %d: %v\n", userID, err)
			}
		}
	} else {
		// Payment failed
		transaction.Status = "failed"
		transaction.ErrorMessage = "Payment declined by processor"
		order.PaymentStatus = "failed"
	}

	// Save transaction
	if err := database.DB.Create(&transaction).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save transaction"})
	}

	// Update order
	if err := database.DB.Save(&order).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update order"})
	}

	// Reload order with items
	if err := database.DB.Preload("Items").First(&order, order.ID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load order details"})
	}

	if !paymentSuccess {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "Payment failed",
			"order":   order,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"order":   order,
		"message": "Payment processed successfully",
	})
}

// GetMyOrders returns every order of the authenticated user, newest first, with
// their items preloaded. The response is a bare JSON array.
func GetMyOrders(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not authenticated"})
	}

	var orders []models.Order
	if err := database.DB.Where("user_id = ?", userID).
		Preload("Items").
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch orders"})
	}

	return c.JSON(orders)
}

// GetOrderByID returns one order with its items. Orders of other users are
// reported as 404 so the endpoint cannot be used to enumerate foreign orders.
func GetOrderByID(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not authenticated"})
	}
	orderID := c.Params("id")

	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, userID).
		Preload("Items").
		First(&order).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}

	return c.JSON(order)
}

// CancelOrder cancels one of the caller's orders.
//
// Only "pending" and "processing" orders can be cancelled; anything further
// along the pipeline returns 400. Cancelling a paid order also marks it
// "refunded" and puts the reserved stock back into the sizes table. The Stripe
// refund call is still a TODO.
func CancelOrder(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not authenticated"})
	}
	orderID := c.Params("id")

	var order models.Order
	if err := database.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}

	// Can only cancel pending or processing orders
	if order.Status != "pending" && order.Status != "processing" {
		return c.Status(400).JSON(fiber.Map{"error": "Cannot cancel order in current status"})
	}

	// If paid, we would need to initiate refund with Stripe
	if order.PaymentStatus == "paid" {
		// TODO: Integrate with Stripe refund API
		order.PaymentStatus = "refunded"

		// Restore stock
		if err := restoreProductStock(&order); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to restore stock"})
		}
	}

	order.Status = "cancelled"
	if err := database.DB.Save(&order).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to cancel order"})
	}

	return c.JSON(fiber.Map{
		"message": "Order cancelled successfully",
		"order":   order,
	})
}

// Helper functions

// generateOrderNumber builds a human readable, collision-resistant order number
// of the form ORD-<unix seconds>-<6 random digits>.
func generateOrderNumber() string {
	timestamp := time.Now().Unix()
	random := rand.Intn(999999)
	return fmt.Sprintf("ORD-%d-%06d", timestamp, random)
}

// simulatePaymentProcessing stands in for the Stripe API.
//
// A payment intent whose last ten characters are the well known Stripe decline
// suffix "0000000002" is always declined; every other intent succeeds. The
// outcome is deliberately deterministic so the demo behaves predictably and the
// payment flow can be tested; swap this function for a real Stripe call later.
func simulatePaymentProcessing(paymentIntentID string) bool {
	// Check if it's a test decline payment intent
	if len(paymentIntentID) > 10 && paymentIntentID[len(paymentIntentID)-10:] == "0000000002" {
		return false // Simulate decline
	}

	return true
}

// decreaseProductStock subtracts the ordered quantity from every size variant of
// the order, guarded by `stock >= quantity` so a concurrent order can never
// push stock below zero. Items without a size ID (for example gift cards) are
// skipped.
//
// Items are loaded into a local slice when the caller has not preloaded them, so
// the passed order (and any unsaved changes on it) is never overwritten.
func decreaseProductStock(order *models.Order) error {
	items, err := orderItems(order)
	if err != nil {
		return err
	}

	// Decrease stock for each item
	for _, item := range items {
		if item.SizeID == nil {
			continue
		}
		if err := database.DB.Exec(`
			UPDATE sizes 
			SET stock = stock - ? 
			WHERE id = ? AND stock >= ?
		`, item.Quantity, *item.SizeID, item.Quantity).Error; err != nil {
			return err
		}
	}

	return nil
}

// restoreProductStock gives back the quantity of every size variant of a
// cancelled order. It is the counterpart of decreaseProductStock and is used
// when a paid order is refunded.
func restoreProductStock(order *models.Order) error {
	items, err := orderItems(order)
	if err != nil {
		return err
	}

	// Restore stock for each item
	for _, item := range items {
		if item.SizeID == nil {
			continue
		}
		if err := database.DB.Exec(`
			UPDATE sizes 
			SET stock = stock + ? 
			WHERE id = ?
		`, item.Quantity, *item.SizeID).Error; err != nil {
			return err
		}
	}

	return nil
}

// orderItems returns the items of an order, reading them from the database when
// the caller passes an order without preloaded items.
func orderItems(order *models.Order) ([]models.OrderItem, error) {
	if len(order.Items) > 0 {
		return order.Items, nil
	}

	var items []models.OrderItem
	if err := database.DB.Where("order_id = ?", order.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// clearUserCart empties the cart of one user once their payment succeeded. The
// cart table is owned by the cart service and only accessed through the shared
// database, so the delete runs as raw SQL.
func clearUserCart(userID uint) error {
	return database.DB.Exec("DELETE FROM cart_items WHERE user_id = ?", userID).Error
}
