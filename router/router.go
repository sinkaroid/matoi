// Package router configures the application routing and middleware.
package router

import (
	"matoi/config"
	"matoi/handlers"
	"matoi/middleware"

	"github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

// SetupRoutes registers all routes for the application, accepting handlers via dependency injection.
func SetupRoutes(app *fiber.App, cfg *config.Config, rule34Handler *handlers.Rule34Handler) {
	// Enable CORS globally
	app.Use(cors.New())

	// Request Logger Middleware
	if cfg.EnableLogs {
		app.Use(logger.New(logger.Config{
			Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} ${url} | ${locals:source}${error}\n",
			TimeFormat: "15:04:05",
			TimeZone:   "Local",
		}))
	}
	// Serve Swagger UI documentation
	app.Use(swaggerui.New(swaggerui.Config{
		FilePath: "./docs/swagger.json",
		Path:     "swagger",
		Title:    "Matoi API Documentation",
	}))

	// Register Home endpoint
	app.Get("/", handlers.HomeHandler)

	// Register Ping endpoint
	app.Get("/ping", handlers.PingHandler)

	// Initialize Authentication Middleware
	authMiddleware := middleware.RequireAPIKey(cfg.APIKey)

	// Register Public Media Proxy endpoint (unprotected)
	app.Get("/api/rule34/media", rule34Handler.ProxyMedia)

	// Protected API Group
	api := app.Group("/api", authMiddleware)

	// Register Rule34 protected endpoints
	api.Get("/rule34/posts", rule34Handler.GetPosts)
}
