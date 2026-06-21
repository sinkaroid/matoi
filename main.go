// Package main is the entry point for the Matoi API service.
//
//	@title			Matoi API Wrapper
//	@version		1.0.2-alpha
//	@description	REST API wrapper/binding for imageboard APIs (Danbooru, Gelbooru, Rule34).
//	@host			localhost:3000
//	@BasePath		/
package main

import (
	"errors"
	"log"
	"matoi/cache"
	"matoi/config"
	"matoi/handlers"
	"matoi/providers"
	"matoi/router"

	"github.com/gofiber/fiber/v3"
)

// ErrorResponse defines the JSON response schema for error responses.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason"`
}

func main() {
	// Load environment variables config
	cfg := config.LoadConfig()

	// Initialize Redis cache client
	log.Println("Initializing Redis connection...")
	if err := cache.InitRedis(cfg.RedisURL); err != nil {
		log.Fatalf("Failed to initialize Redis cache: %v", err)
	}
	log.Println("Redis connected successfully.")

	// Instantiate Providers
	rule34Provider := providers.NewRule34Provider(cfg)

	// Instantiate Handlers
	rule34Handler := handlers.NewRule34Handler(rule34Provider)

	// Initialize Go Fiber app with a custom global error handler
	app := fiber.New(fiber.Config{
		AppName: "Matoi API Wrapper",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := err.Error()

			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
				message = e.Message
			}

			return c.Status(code).JSON(ErrorResponse{
				Success: false,
				Reason:  message,
			})
		},
	})

	// Setup application routing with dependency injection
	router.SetupRoutes(app, cfg, rule34Handler)

	// Run Fiber server on configured port
	log.Printf("Starting server on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to run the server: %v", err)
	}
}
