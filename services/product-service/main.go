package main

import (
	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/product-service/database"
	"github.com/adibfahimi/moda-style/services/product-service/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	database.Connect()
	database.Migrate()

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// API v1 routes
	api := app.Group("/api/v1")

	// Public product routes
	api.Get("/products", handlers.ListProducts)
	api.Get("/products/:id", handlers.GetProduct)
	api.Get("/categories", handlers.ListCategories)
	api.Get("/products/:id/reviews", handlers.GetProductReviews)

	// Protected review routes (require authentication)
	api.Post("/products/:id/reviews", common.RequireAuth, handlers.CreateReview)

	app.Listen(":8002")
}
