package main

import (
	"github.com/adibfahimi/moda-style/common"
	"github.com/adibfahimi/moda-style/services/auth-service/database"
	"github.com/adibfahimi/moda-style/services/auth-service/handlers"
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

	auth := api.Group("/auth")
	auth.Post("/register", handlers.Register)
	auth.Post("/login", handlers.Login)
	auth.Get("/profile", common.RequireAuth, handlers.GetProfile)
	auth.Patch("/profile", common.RequireAuth, handlers.UpdateProfile)
	auth.Post("/reset-password", handlers.ResetPassword)

	app.Listen(":8001")
}
