package main

import (
	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/cart-service/database"
	"github.com/adibfahimi/moda-style/services/cart-service/handlers"

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

	api := app.Group("/api/v1")

	api.Get("/cart", common.RequireAuth, handlers.GetCart)
	api.Post("/cart", common.RequireAuth, handlers.AddToCart)
	api.Put("/cart/:id", common.RequireAuth, handlers.UpdateCartItem)
	api.Delete("/cart/:id", common.RequireAuth, handlers.RemoveFromCart)
	api.Delete("/cart", common.RequireAuth, handlers.ClearCart)

	api.Get("/wishlist", common.RequireAuth, handlers.GetWishlist)
	api.Post("/wishlist/:product_id", common.RequireAuth, handlers.ToggleWishlistItem)

	app.Listen(":8003")
}
