// Package handlers contains all the HTTP endpoints handlers for the application.
package handlers

import (
	"github.com/gofiber/fiber/v3"
)

// PingResponse defines the JSON response schema for ping.
type PingResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
}

// PingHandler responds with a simple status OK.
//
//	@Summary	Ping the server
//	@Tags		system
//	@Produce	json
//	@Success	200	{object}	PingResponse	"Server is running"
//	@Failure	500	{object}	map[string]string		"Internal Server Error"
//	@Router		/ping [get]
func PingHandler(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(PingResponse{
		Success: true,
		Status:  "ok",
	})
}
