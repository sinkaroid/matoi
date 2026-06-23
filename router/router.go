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
func SetupRoutes(app *fiber.App, cfg *config.Config, rule34Handler *handlers.Rule34Handler, danbooruHandler *handlers.DanbooruHandler, gelbooruHandler *handlers.GelbooruHandler, tbibHandler *handlers.TbibHandler, xbooruHandler *handlers.XbooruHandler,
	hypnohubHandler *handlers.HypnohubHandler,
	safebooruHandler *handlers.SafebooruHandler,
	yandereHandler *handlers.YandereHandler,
	konachanComHandler *handlers.KonachanComHandler,
	konachanNetHandler *handlers.KonachanNetHandler,
	e621Handler *handlers.E621Handler,
	e926Handler *handlers.E926Handler,
	furbooruHandler *handlers.FurbooruHandler,
	derpibooruHandler *handlers.DerpibooruHandler,
	graphqlHandler *handlers.GraphQLHandler,
) {
	// Enable CORS globally
	app.Use(cors.New())

	// Request Logger Middleware
	if cfg.EnableLogs {
		app.Use(logger.New(logger.Config{
			Format:     "${time} | ${status} | ${latency} | ${ip} | ${ua} | ${method} ${url} | ${locals:source}${error}\n",
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
	app.Get("/api/danbooru/media", danbooruHandler.ProxyMedia)
	app.Get("/api/gelbooru/media", gelbooruHandler.ProxyMedia)
	app.Get("/api/tbib/media", tbibHandler.ProxyMedia)
	app.Get("/api/xbooru/media", xbooruHandler.ProxyMedia)
	app.Get("/api/hypnohub/media", hypnohubHandler.ProxyMedia)
	app.Get("/api/safebooru/media", safebooruHandler.ProxyMedia)
	app.Get("/api/yandere/media", yandereHandler.ProxyMedia)
	app.Get("/api/konachan_com/media", konachanComHandler.ProxyMedia)
	app.Get("/api/konachan_net/media", konachanNetHandler.ProxyMedia)
	app.Get("/api/e621/media", e621Handler.ProxyMedia)
	app.Get("/api/e926/media", e926Handler.ProxyMedia)
	app.Get("/api/furbooru/media", furbooruHandler.ProxyMedia)
	app.Get("/api/derpibooru/media", derpibooruHandler.ProxyMedia)

	// Protected API Group
	api := app.Group("/api", authMiddleware)

	// Register Rule34 protected endpoints
	api.Get("/rule34/posts", rule34Handler.GetPosts)
	api.Get("/rule34/query_completion", rule34Handler.QueryCompletion)

	// Register Danbooru protected endpoints
	api.Get("/danbooru/posts", danbooruHandler.GetPosts)
	api.Get("/danbooru/query_completion", danbooruHandler.QueryCompletion)

	// Register Gelbooru protected endpoints
	api.Get("/gelbooru/posts", gelbooruHandler.GetPosts)
	api.Get("/gelbooru/query_completion", gelbooruHandler.QueryCompletion)

	// Register TBIB protected endpoints
	api.Get("/tbib/posts", tbibHandler.GetPosts)
	api.Get("/tbib/query_completion", tbibHandler.QueryCompletion)

	// Register Xbooru protected endpoints
	api.Get("/xbooru/posts", xbooruHandler.GetPosts)
	api.Get("/xbooru/query_completion", xbooruHandler.QueryCompletion)

	// Register Hypnohub protected endpoints
	api.Get("/hypnohub/posts", hypnohubHandler.GetPosts)
	api.Get("/hypnohub/query_completion", hypnohubHandler.QueryCompletion)

	// Register Safebooru protected endpoints
	api.Get("/safebooru/posts", safebooruHandler.GetPosts)
	api.Get("/safebooru/query_completion", safebooruHandler.QueryCompletion)

	// Register Yandere protected endpoints
	api.Get("/yandere/posts", yandereHandler.GetPosts)
	api.Get("/yandere/query_completion", yandereHandler.QueryCompletion)

	// Register Konachan protected endpoints
	api.Get("/konachan_com/posts", konachanComHandler.GetPosts)
	api.Get("/konachan_com/query_completion", konachanComHandler.QueryCompletion)
	api.Get("/konachan_net/posts", konachanNetHandler.GetPosts)
	api.Get("/konachan_net/query_completion", konachanNetHandler.QueryCompletion)

	// Register E621 protected endpoints
	api.Get("/e621/posts", e621Handler.GetPosts)
	api.Get("/e621/query_completion", e621Handler.QueryCompletion)

	// Register E926 protected endpoints
	api.Get("/e926/posts", e926Handler.GetPosts)
	api.Get("/e926/query_completion", e926Handler.QueryCompletion)

	// Register Furbooru protected endpoints
	api.Get("/furbooru/posts", furbooruHandler.GetPosts)
	api.Get("/furbooru/query_completion", furbooruHandler.QueryCompletion)

	// Register Derpibooru protected endpoints
	api.Get("/derpibooru/posts", derpibooruHandler.GetPosts)
	api.Get("/derpibooru/query_completion", derpibooruHandler.QueryCompletion)

	// Register GraphQL endpoints
	if cfg.EnableGraphQL && graphqlHandler != nil {
		app.Get("/graphql", graphqlHandler.Playground) // Unprotected UI
		api.Post("/graphql", graphqlHandler.Handle)    // Protected API
	}
}
