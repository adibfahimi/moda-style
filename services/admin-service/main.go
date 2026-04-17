package main

import (
	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/admin-service/database"
	"github.com/adibfahimi/moda-style/services/admin-service/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	database.Connect()
	database.Migrate()

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return common.SendErrorResponse(c, fiber.StatusInternalServerError, err.Error())
		},
	})

	// Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "admin"})
	})

	// Serve uploaded files
	app.Static("/api/v1/admin/uploads", "./uploads")
	app.Static("/api/admin/uploads", "./uploads")

	api := app.Group("/api/v1")

	// Admin routes - all require authentication and admin role
	admin := api.Group("/admin", common.RequireAuth, common.RequireAdmin)

	// Dashboard & Analytics
	admin.Get("/dashboard/stats", handlers.GetDashboardStats)
	admin.Get("/dashboard/activity", handlers.GetRecentActivity)
	admin.Get("/dashboard/logs", handlers.GetActivityLogs)

	// User Management
	admin.Get("/users", handlers.GetUsers)
	admin.Get("/users/analytics", handlers.GetUserAnalytics)
	admin.Get("/users/:id", handlers.GetUserDetails)
	admin.Put("/users/:id", handlers.UpdateUser)
	admin.Delete("/users/:id", handlers.DeleteUser)
	admin.Post("/users/:id/ban", handlers.BanUser)
	admin.Post("/users/:id/unban", handlers.UnbanUser)

	// Product Management
	admin.Get("/products", handlers.GetProductStats)
	admin.Get("/products/images", handlers.ListProductImages)
	admin.Post("/products/upload-image", handlers.UploadProductImage)
	admin.Post("/products", handlers.CreateProduct)
	admin.Put("/products/:id", handlers.UpdateProduct)
	admin.Delete("/products/:id", handlers.DeleteProduct)

	// Product Size/Variant Management
	admin.Get("/products/:id/sizes", handlers.GetProductSizes)
	admin.Post("/products/:id/sizes", handlers.AddProductSize)
	admin.Put("/products/:id/sizes/:sizeId", handlers.UpdateProductSize)
	admin.Delete("/products/:id/sizes/:sizeId", handlers.DeleteProductSize)

	// Category Management
	admin.Get("/categories", handlers.GetCategories)
	admin.Post("/categories", handlers.CreateCategory)
	admin.Put("/categories/:id", handlers.UpdateCategory)
	admin.Delete("/categories/:id", handlers.DeleteCategory)

	// Order Management
	admin.Get("/orders", handlers.GetOrders)
	admin.Get("/orders/analytics", handlers.GetOrderAnalytics)
	admin.Get("/orders/:id", handlers.GetOrderDetails)
	admin.Put("/orders/:id", handlers.UpdateOrderStatus)
	admin.Delete("/orders/:id", handlers.DeleteOrder)

	app.Listen(":8004")
}
