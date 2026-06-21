// Package main is the entry point for the Matoi API service.
//
//	@title			sinkaroid/matoi
//	@description	REST API wrapper/binding for imageboard APIs (Danbooru, Gelbooru, Rule34).
//	@tag.name		graphql
//	@tag.name		danbooru
//	@tag.name		gelbooru
//	@tag.name		rule34
//	@tag.name		tbib
//	@tag.name		xbooru
//	@tag.name		hypnohub
//	@tag.name		safebooru
//	@tag.name		yandere
//	@tag.name		system
//	@BasePath		/
//	@securityDefinitions.apikey ApiKeyAuth
//	@in query
//	@name api_key
//	@description Provide the API key via the `api_key` query parameter or `Authorization: Bearer <key>` header.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

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
	versionFlag := flag.Bool("version", false, "Print the application version")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("sinkaroid/matoi v%s\n", config.AppVersion)
		os.Exit(0)
	}

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
	danbooruProvider := providers.NewDanbooruProvider(cfg)
	gelbooruProvider := providers.NewGelbooruProvider(cfg)
	tbibProvider := providers.NewTbibProvider(cfg)
	xbooruProvider := providers.NewXbooruProvider(cfg)
	hypnohubProvider := providers.NewHypnohubProvider(cfg)
	safebooruProvider := providers.NewSafebooruProvider(cfg)
	yandereProvider := providers.NewYandereProvider(cfg)

	// Instantiate Handlers
	rule34Handler := handlers.NewRule34Handler(rule34Provider)
	danbooruHandler := handlers.NewDanbooruHandler(danbooruProvider)
	gelbooruHandler := handlers.NewGelbooruHandler(gelbooruProvider)
	tbibHandler := handlers.NewTbibHandler(tbibProvider)
	xbooruHandler := handlers.NewXbooruHandler(xbooruProvider)
	hypnohubHandler := handlers.NewHypnohubHandler(hypnohubProvider)
	safebooruHandler := handlers.NewSafebooruHandler(safebooruProvider)
	yandereHandler := handlers.NewYandereHandler(yandereProvider)

	var graphqlHandler *handlers.GraphQLHandler
	if cfg.EnableGraphQL {
		graphqlHandler = handlers.NewGraphQLHandler(cfg, danbooruProvider, gelbooruProvider, rule34Provider, tbibProvider, xbooruProvider, hypnohubProvider, safebooruProvider, yandereProvider)
	}

	// Initialize Go Fiber app with a custom global error handler
	app := fiber.New(fiber.Config{
		AppName: "sinkaroid/matoi",
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
	router.SetupRoutes(app, cfg, rule34Handler, danbooruHandler, gelbooruHandler, tbibHandler, xbooruHandler, hypnohubHandler, safebooruHandler, yandereHandler, graphqlHandler)

	// Run Fiber server on configured port
	log.Printf("Starting server on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to run the server: %v", err)
	}
}
