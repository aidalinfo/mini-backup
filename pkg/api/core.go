package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

var source_api = "API CORE"

// NewServer configure et retourne un serveur Fiber.
func ApiServer() *fiber.App {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))
	// Configurer les routes
	SetupRoutes(app)

	return app
}
