package handlers

import (
	"fmt"
	"strconv"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/admin-service/database"
	"github.com/adibfahimi/moda-style/services/admin-service/models"
	"github.com/gofiber/fiber/v2"
)

// GetOrders retrieves paginated list of orders with stats
func GetOrders(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	status := c.Query("status")
	paymentStatus := c.Query("payment_status")
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	query := `
		SELECT 
			o.id,
			o.order_number,
			o.user_id,
			u.name as user_name,
			u.email as user_email,
			o.status,
			o.payment_status,
			o.payment_method,
			o.total_amount,
			o.shipping_address,
			o.notes,
			o.created_at,
			o.updated_at,
			COUNT(oi.id) as item_count
		FROM orders o
		LEFT JOIN users u ON o.user_id = u.id
		LEFT JOIN order_items oi ON o.id = oi.order_id
		WHERE o.deleted_at IS NULL
	`

	countQuery := `
		SELECT COUNT(DISTINCT o.id)
		FROM orders o
		LEFT JOIN users u ON o.user_id = u.id
		WHERE o.deleted_at IS NULL
	`

	args := []interface{}{}
	countArgs := []interface{}{}

	if status != "" {
		query += " AND o.status = ?"
		countQuery += " AND o.status = ?"
		args = append(args, status)
		countArgs = append(countArgs, status)
	}

	if paymentStatus != "" {
		query += " AND o.payment_status = ?"
		countQuery += " AND o.payment_status = ?"
		args = append(args, paymentStatus)
		countArgs = append(countArgs, paymentStatus)
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query += " AND (o.order_number LIKE ? OR u.name LIKE ? OR u.email LIKE ?)"
		countQuery += " AND (o.order_number LIKE ? OR u.name LIKE ? OR u.email LIKE ?)"
		args = append(args, searchPattern, searchPattern, searchPattern)
		countArgs = append(countArgs, searchPattern, searchPattern, searchPattern)
	}

	query += " GROUP BY o.id, u.name, u.email ORDER BY o.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	var orders []models.OrderStats
	if err := database.DB.Raw(query, args...).Scan(&orders).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch orders")
	}

	var total int64
	if err := database.DB.Raw(countQuery, countArgs...).Scan(&total).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to count orders")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GetOrderDetails retrieves detailed information about a specific order
func GetOrderDetails(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid order ID")
	}

	var order models.Order
	if err := database.DB.Preload("Items").First(&order, id).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Order not found")
	}

	// Get user info
	var orderStats models.OrderStats
	query := `
		SELECT 
			o.id,
			o.order_number,
			o.user_id,
			u.name as user_name,
			u.email as user_email,
			o.status,
			o.payment_status,
			o.payment_method,
			o.total_amount,
			o.shipping_address,
			o.notes,
			o.created_at,
			o.updated_at,
			COUNT(oi.id) as item_count
		FROM orders o
		LEFT JOIN users u ON o.user_id = u.id
		LEFT JOIN order_items oi ON o.id = oi.order_id
		WHERE o.id = ? AND o.deleted_at IS NULL
		GROUP BY o.id, u.name, u.email
	`

	if err := database.DB.Raw(query, id).Scan(&orderStats).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Order not found")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"order": orderStats,
		"items": order.Items,
	})
}

// UpdateOrderStatus updates the status of an order
func UpdateOrderStatus(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid order ID")
	}

	var req struct {
		Status        string `json:"status"`
		PaymentStatus string `json:"payment_status"`
		Notes         string `json:"notes"`
	}

	if err := c.BodyParser(&req); err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Validate status
	validStatuses := map[string]bool{
		"pending": true, "processing": true, "shipped": true,
		"delivered": true, "cancelled": true,
	}
	if req.Status != "" && !validStatuses[req.Status] {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid order status")
	}

	validPaymentStatuses := map[string]bool{
		"pending": true, "paid": true, "failed": true, "refunded": true,
	}
	if req.PaymentStatus != "" && !validPaymentStatuses[req.PaymentStatus] {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid payment status")
	}

	var order models.Order
	if err := database.DB.First(&order, id).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Order not found")
	}

	updates := make(map[string]interface{})
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.PaymentStatus != "" {
		updates["payment_status"] = req.PaymentStatus
	}
	if req.Notes != "" {
		updates["notes"] = req.Notes
	}

	if err := database.DB.Model(&order).Updates(updates).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to update order")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "updated", "order", uint(id),
		fmt.Sprintf("Updated order %s status", order.OrderNumber), c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Order updated successfully",
		"order":   order,
	})
}

// DeleteOrder soft deletes an order
func DeleteOrder(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid order ID")
	}

	var order models.Order
	if err := database.DB.First(&order, id).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "Order not found")
	}

	if err := database.DB.Delete(&order).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete order")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "deleted", "order", uint(id),
		fmt.Sprintf("Deleted order %s", order.OrderNumber), c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "Order deleted successfully",
	})
}

// GetOrderAnalytics returns order analytics for dashboard
func GetOrderAnalytics(c *fiber.Ctx) error {
	var analytics struct {
		TotalOrders      int64   `json:"total_orders"`
		PendingOrders    int64   `json:"pending_orders"`
		ProcessingOrders int64   `json:"processing_orders"`
		ShippedOrders    int64   `json:"shipped_orders"`
		DeliveredOrders  int64   `json:"delivered_orders"`
		CancelledOrders  int64   `json:"cancelled_orders"`
		TotalRevenue     float64 `json:"total_revenue"`
		PendingRevenue   float64 `json:"pending_revenue"`
		PaidRevenue      float64 `json:"paid_revenue"`
		OrdersToday      int64   `json:"orders_today"`
		OrdersThisWeek   int64   `json:"orders_this_week"`
		OrdersThisMonth  int64   `json:"orders_this_month"`
	}

	// Total orders
	database.DB.Model(&models.Order{}).Count(&analytics.TotalOrders)

	// Orders by status
	database.DB.Model(&models.Order{}).Where("status = ?", "pending").Count(&analytics.PendingOrders)
	database.DB.Model(&models.Order{}).Where("status = ?", "processing").Count(&analytics.ProcessingOrders)
	database.DB.Model(&models.Order{}).Where("status = ?", "shipped").Count(&analytics.ShippedOrders)
	database.DB.Model(&models.Order{}).Where("status = ?", "delivered").Count(&analytics.DeliveredOrders)
	database.DB.Model(&models.Order{}).Where("status = ?", "cancelled").Count(&analytics.CancelledOrders)

	// Revenue
	database.DB.Model(&models.Order{}).Select("COALESCE(SUM(total_amount), 0)").Scan(&analytics.TotalRevenue)
	database.DB.Model(&models.Order{}).Where("payment_status = ?", "pending").Select("COALESCE(SUM(total_amount), 0)").Scan(&analytics.PendingRevenue)
	database.DB.Model(&models.Order{}).Where("payment_status = ?", "paid").Select("COALESCE(SUM(total_amount), 0)").Scan(&analytics.PaidRevenue)

	// Orders by time period
	database.DB.Model(&models.Order{}).Where("DATE(created_at) = CURRENT_DATE").Count(&analytics.OrdersToday)
	database.DB.Model(&models.Order{}).Where("created_at >= NOW() - INTERVAL '7 days'").Count(&analytics.OrdersThisWeek)
	database.DB.Model(&models.Order{}).Where("created_at >= NOW() - INTERVAL '30 days'").Count(&analytics.OrdersThisMonth)

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"analytics": analytics,
	})
}
