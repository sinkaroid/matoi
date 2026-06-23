package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"method", "route", "status"},
	)

	httpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed.",
		},
		[]string{"method"},
	)
)

// PrometheusMiddleware intercepts requests to gather HTTP metrics.
func PrometheusMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		method := c.Method()
		httpRequestsInFlight.WithLabelValues(method).Inc()
		defer httpRequestsInFlight.WithLabelValues(method).Dec()

		start := time.Now()

		err := c.Next()

		status := fiber.StatusInternalServerError
		if err != nil {
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
			}
		} else {
			status = c.Response().StatusCode()
		}

		route := c.Route().Path
		if route == "" {
			route = "404 Not Found"
			status = fiber.StatusNotFound
		}

		// Prevent tracking the /metrics endpoint itself to avoid noise
		if route == "/metrics" {
			return err
		}

		statusStr := strconv.Itoa(status)
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(method, route, statusStr).Inc()
		httpRequestDuration.WithLabelValues(method, route, statusStr).Observe(duration)

		return err
	}
}
