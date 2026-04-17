package handlers

import (
	"strconv"

	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/admin-service/database"
	"github.com/adibfahimi/moda-style/services/admin-service/models"
	"github.com/gofiber/fiber/v2"
)

// User represents the user model for admin operations
type User struct {
	ID    uint   `json:"id"`
	Name  string `json:"name" validate:"required,min=2"`
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=user admin"`
}

// GetUsers returns a paginated list of users with statistics
func GetUsers(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Filter parameters
	role := c.Query("role")
	search := c.Query("search")

	var users []models.UserStats
	var total int64

	query := database.DB.Table("users").
		Select(`
			users.id,
			users.name,
			users.email,
			users.role,
			users.banned,
			COUNT(DISTINCT reviews.id) as review_count,
			COUNT(DISTINCT wishlist_items.id) as wishlist_count,
			users.created_at
		`).
		Joins("LEFT JOIN reviews ON users.id = reviews.user_id AND reviews.deleted_at IS NULL").
		Joins("LEFT JOIN wishlist_items ON users.id = wishlist_items.user_id AND wishlist_items.deleted_at IS NULL").
		Where("users.deleted_at IS NULL")

	// Apply filters
	if role != "" {
		query = query.Where("users.role = ?", role)
	}

	if search != "" {
		query = query.Where("users.name ILIKE ? OR users.email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Count total with filters
	countQuery := database.DB.Table("users").Where("deleted_at IS NULL")
	if role != "" {
		countQuery = countQuery.Where("role = ?", role)
	}
	if search != "" {
		countQuery = countQuery.Where("name ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	countQuery.Count(&total)

	// Get paginated results
	query.Group("users.id").
		Order("users.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&users)

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUserDetails returns detailed information about a specific user
func GetUserDetails(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid user ID")
	}

	var user models.UserStats

	if err := database.DB.Table("users").
		Select(`
			users.id,
			users.name,
			users.email,
			users.role,
			users.banned,
			COUNT(DISTINCT reviews.id) as review_count,
			COUNT(DISTINCT wishlist_items.id) as wishlist_count,
			users.created_at
		`).
		Joins("LEFT JOIN reviews ON users.id = reviews.user_id AND reviews.deleted_at IS NULL").
		Joins("LEFT JOIN wishlist_items ON users.id = wishlist_items.user_id AND wishlist_items.deleted_at IS NULL").
		Where("users.id = ? AND users.deleted_at IS NULL", id).
		Group("users.id").
		Scan(&user).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "User not found")
	}

	// Get user's recent reviews
	type ReviewInfo struct {
		ID          uint   `json:"id"`
		ProductID   uint   `json:"product_id"`
		ProductName string `json:"product_name"`
		Rating      int    `json:"rating"`
		Comment     string `json:"comment"`
		CreatedAt   string `json:"created_at"`
	}

	var reviews []ReviewInfo
	database.DB.Table("reviews").
		Select("reviews.id, reviews.product_id, products.name as product_name, reviews.rating, reviews.comment, reviews.created_at").
		Joins("LEFT JOIN products ON reviews.product_id = products.id").
		Where("reviews.user_id = ? AND reviews.deleted_at IS NULL", id).
		Order("reviews.created_at DESC").
		Limit(10).
		Scan(&reviews)

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"user":    user,
		"reviews": reviews,
	})
}

// UpdateUser updates user information
func UpdateUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid user ID")
	}

	var req User
	if !common.ParseAndValidate(c, &req) {
		return nil
	}

	// Check if user exists
	var existingUser User
	if err := database.DB.Table("users").Where("id = ? AND deleted_at IS NULL", id).Scan(&existingUser).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "User not found")
	}

	// Check if email is already taken by another user
	if req.Email != "" && req.Email != existingUser.Email {
		var emailExists bool
		database.DB.Table("users").
			Select("count(*) > 0").
			Where("email = ? AND id != ? AND deleted_at IS NULL", req.Email, id).
			Scan(&emailExists)

		if emailExists {
			return common.SendErrorResponse(c, fiber.StatusConflict, "Email already in use")
		}
	}

	// Update user
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}

	if err := database.DB.Table("users").Where("id = ?", id).Updates(updates).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to update user")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "updated", "user", uint(id), "Updated user: "+req.Name, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "User updated successfully",
	})
}

// DeleteUser soft deletes a user
func DeleteUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid user ID")
	}

	// Check if user exists
	var userName string
	if err := database.DB.Table("users").
		Select("name").
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(&userName).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "User not found")
	}

	// Prevent deleting yourself
	currentUserID := c.Locals("userID").(uint)
	if currentUserID == uint(id) {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Cannot delete your own account")
	}

	// Soft delete user
	if err := database.DB.Table("users").Where("id = ?", id).Update("deleted_at", "NOW()").Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to delete user")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "deleted", "user", uint(id), "Deleted user: "+userName, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "User deleted successfully",
	})
}

// BanUser permanently bans a user
func BanUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid user ID")
	}

	// Check if user exists and get details
	var user struct {
		Name   string `json:"name"`
		Banned bool   `json:"banned"`
	}
	if err := database.DB.Table("users").
		Select("name, banned").
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(&user).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "User not found")
	}

	// Prevent banning yourself
	currentUserID := c.Locals("userID").(uint)
	if currentUserID == uint(id) {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Cannot ban your own account")
	}

	// Check if already banned
	if user.Banned {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "User is already banned")
	}

	// Ban user
	if err := database.DB.Table("users").Where("id = ?", id).Update("banned", true).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to ban user")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "banned", "user", uint(id), "Permanently banned user: "+user.Name, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "User banned successfully",
	})
}

// UnbanUser removes the ban from a user
func UnbanUser(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "Invalid user ID")
	}

	// Check if user exists and get details
	var user struct {
		Name   string `json:"name"`
		Banned bool   `json:"banned"`
	}
	if err := database.DB.Table("users").
		Select("name, banned").
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(&user).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusNotFound, "User not found")
	}

	// Check if not banned
	if !user.Banned {
		return common.SendErrorResponse(c, fiber.StatusBadRequest, "User is not banned")
	}

	// Unban user
	if err := database.DB.Table("users").Where("id = ?", id).Update("banned", false).Error; err != nil {
		return common.SendErrorResponse(c, fiber.StatusInternalServerError, "Failed to unban user")
	}

	// Log activity
	adminID := c.Locals("userID").(uint)
	adminName := c.Locals("userName").(string)
	LogActivity(adminID, adminName, "unbanned", "user", uint(id), "Unbanned user: "+user.Name, c.IP())

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "User unbanned successfully",
	})
}

// GetUserAnalytics returns user-related analytics
func GetUserAnalytics(c *fiber.Ctx) error {
	type Analytics struct {
		TotalUsers       int64 `json:"total_users"`
		AdminUsers       int64 `json:"admin_users"`
		RegularUsers     int64 `json:"regular_users"`
		NewUsersToday    int64 `json:"new_users_today"`
		NewUsersThisWeek int64 `json:"new_users_this_week"`
		ActiveReviewers  int64 `json:"active_reviewers"`
	}

	var analytics Analytics

	// Total users
	database.DB.Table("users").Where("deleted_at IS NULL").Count(&analytics.TotalUsers)

	// Admin users
	database.DB.Table("users").Where("role = ? AND deleted_at IS NULL", "admin").Count(&analytics.AdminUsers)

	// Regular users
	database.DB.Table("users").Where("role = ? AND deleted_at IS NULL", "user").Count(&analytics.RegularUsers)

	// New users today
	database.DB.Table("users").
		Where("DATE(created_at) = CURRENT_DATE AND deleted_at IS NULL").
		Count(&analytics.NewUsersToday)

	// New users this week
	database.DB.Table("users").
		Where("created_at >= CURRENT_DATE - INTERVAL '7 days' AND deleted_at IS NULL").
		Count(&analytics.NewUsersThisWeek)

	// Active reviewers (users with at least one review)
	database.DB.Table("users").
		Joins("INNER JOIN reviews ON users.id = reviews.user_id AND reviews.deleted_at IS NULL").
		Where("users.deleted_at IS NULL").
		Distinct("users.id").
		Count(&analytics.ActiveReviewers)

	return common.SendSuccessResponse(c, fiber.StatusOK, fiber.Map{
		"analytics": analytics,
	})
}
