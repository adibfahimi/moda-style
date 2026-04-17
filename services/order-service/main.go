package main

import (
	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/order-service/database"
	"github.com/adibfahimi/moda-style/services/order-service/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	database.Connect()
	database.Migrate()

	app := fiber.New(fiber.Config{
		AppName: "Moda Style - Order Service",
	})

	// Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Use(logger.New())

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "order-service"})
	})

	// API routes
	api := app.Group("/api/v1")

	// Order routes
	api.Post("/orders", common.RequireAuth, handlers.CreateOrder)
	api.Post("/orders/:id/pay", common.RequireAuth, handlers.ProcessPayment)
	api.Get("/orders/my-orders", common.RequireAuth, handlers.GetMyOrders)
	api.Get("/orders/:id", common.RequireAuth, handlers.GetOrderByID)
	api.Post("/orders/:id/cancel", common.RequireAuth, handlers.CancelOrder)

	// Start server
	app.Listen(":8005")
}
