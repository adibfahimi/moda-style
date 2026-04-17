package handlers

import (
	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/admin-service/database"
	"github.com/adibfahimi/moda-style/services/admin-service/models"
	"github.com/gofiber/fiber/v2"
)

// GetDashboardStats returns overall statistics for the dashboard
func GetDashboardStats(c *fiber.Ctx) error {
	var stats models.DashboardStats

	// Count total users
	database.DB.Table("users").Count(&stats.TotalUsers)

	// Count total products
	database.DB.Table("products").Where("deleted_at IS NULL").Count(&stats.TotalProducts)

	// Count total categories
	database.DB.Table("categories").Where("deleted_at IS NULL").Count(&stats.TotalCategories)

	// Count total reviews
	database.DB.Table("reviews").Where("deleted_at IS NULL").Count(&stats.TotalReviews)

	// Count low stock products (stock < 10)
	database.DB.Table("products").
		Select("COUNT(DISTINCT products.id)").
		Joins("LEFT JOIN sizes ON products.id = sizes.product_id").
		Where("products.deleted_at IS NULL").
		Group("products.id").
		Having("SUM(COALESCE(sizes.stock, 0)) < ?", 10).
		Count(&stats.LowStockProducts)

	// Calculate average rating
	database.DB.Table("reviews").
		Where("deleted_at IS NULL").
		Select("COALESCE(AVG(rating), 0)").
		Scan(&stats.AverageRating)

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"stats": stats,
	})
}

// GetRecentActivity returns recent activities in the system
func GetRecentActivity(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}

	var activities []models.RecentActivity

	// Get recent user registrations
	var userActivities []models.RecentActivity
	database.DB.Table("users").
		Select("id, 'user_registered' as type, CONCAT('New user registered: ', name) as description, id as user_id, name as user_name, created_at").
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit / 4).
		Scan(&userActivities)

	activities = append(activities, userActivities...)

	// Get recent product additions
	var productActivities []models.RecentActivity
	database.DB.Table("products").
		Select("id, 'product_created' as type, CONCAT('New product added: ', name) as description, NULL as user_id, '' as user_name, created_at").
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit / 4).
		Scan(&productActivities)

	activities = append(activities, productActivities...)

	// Get recent reviews
	var reviewActivities []models.RecentActivity
	database.DB.Table("reviews").
		Select("reviews.id, 'review_added' as type, CONCAT(reviews.user_name, ' reviewed a product (', reviews.rating, ' stars)') as description, reviews.user_id, reviews.user_name, reviews.created_at").
		Where("reviews.deleted_at IS NULL").
		Order("reviews.created_at DESC").
		Limit(limit / 2).
		Scan(&reviewActivities)

	activities = append(activities, reviewActivities...)

	// Sort all activities by created_at
	// Note: In production, use a more efficient approach with UNION queries

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"activities": activities,
		"count":      len(activities),
	})
}

// GetActivityLogs returns admin activity logs for audit trail
func GetActivityLogs(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var logs []models.ActivityLog
	var total int64

	query := database.DB.Model(&models.ActivityLog{})

	// Filter by admin ID if provided
	if adminID := c.QueryInt("admin_id", 0); adminID > 0 {
		query = query.Where("admin_id = ?", adminID)
	}

	// Filter by action if provided
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}

	// Filter by resource if provided
	if resource := c.Query("resource"); resource != "" {
		query = query.Where("resource = ?", resource)
	}

	// Count total
	query.Count(&total)

	// Get paginated results
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch activity logs")
	}

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// LogActivity creates a new activity log entry
func LogActivity(adminID uint, adminName, action, resource string, resourceID uint, description, ipAddress string) error {
	log := models.ActivityLog{
		AdminID:     adminID,
		AdminName:   adminName,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Description: description,
		IPAddress:   ipAddress,
	}

	return database.DB.Create(&log).Error
}
