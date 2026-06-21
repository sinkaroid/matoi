// Package middleware provides custom Fiber middlewares for the application.
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// RequireAPIKey creates a middleware that protects endpoints using a static API key.
// It supports passing the key via the "api_key" query parameter or the "Authorization: Bearer <key>" header.
func RequireAPIKey(configuredKey string) fiber.Handler {
	return func(c fiber.Ctx) error {
		// If no key is configured in the environment, we can optionally bypass auth,
		// but typically it's safer to either bypass or strict fail.
		// For a developer-friendly setup, we'll bypass if it's completely empty.
		if configuredKey == "" {
			return c.Next()
		}

		// 1. Check Query Parameter: ?api_key=XYZ
		apiKey := c.Query("api_key")

		// 2. Check Authorization Header: Bearer XYZ
		if apiKey == "" {
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// Compare the provided key with the configured key
		if apiKey != configuredKey {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid or missing API key")
		}

		// Success, continue to the next handler
		return c.Next()
	}
}
