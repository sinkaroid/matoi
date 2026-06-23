# Product Requirement Document (PRD): Universal Telemetry Standardization

| **Topic** | **Standardization of Prometheus Metrics Collection** |
| :--- | :--- |
| **Status** | Active / Implementation |
| **Target Delivery** | 2026-Q3 |
| **Author** | Engineering Team |

---

## 1. Executive Summary

### 1.1 Objective

Establish a standardized metrics collection (telemetry) for all internal and external microservices / API Gateways utilizing three different frameworks: **GoFiber** (Go), **Hono** (TypeScript/Bun/Node), and **ElysiaJS** (TypeScript/Bun).

### 1.2 Success Metrics

- **100% Visibility:** All registered API routes must expose performance metrics without exception.
- **Zero Out-of-Memory (OOM) Surprises:** Early detection of memory spikes via alerting before the OS executes an OOM Kill.
- **Low Overhead:** The Prometheus client implementation must not degrade API throughput by more than 3%.
- **Clean Metrics (No High-Cardinality):** No dynamic data (ID, UUID, Email) leakage into Prometheus labels.

---

## 2. Metrics Architecture & Standard Specification

Every framework **must** expose a `/metrics` endpoint. Metrics are divided into two categories: HTTP Metrics (RED Method) and Runtime/System Metrics (Memory & CPU).

### 2.1 HTTP Metrics Standard (All Frameworks)

| Metric Name | Type | Labels | Description |
| :--- | :--- | :--- | :--- |
| `http_requests_total` | Counter | `method, route, status` | Total number of incoming HTTP requests. |
| `http_request_duration_seconds` | Histogram | `method, route, status` | Request latency. Buckets: `[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]` |
| `http_requests_in_flight` | Gauge | `method` | Number of requests currently being processed (active). |

> **Route Transformation Rule (Anti High-Cardinality):**
> All dynamic routes must be masked before being recorded as Prometheus labels.
>
> - ❌ **Incorrect:** `/api/v1/users/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d`
> - ✅ **Correct:** `/api/v1/users/:id` or `/api/v1/users/{id}`

---

## 3. Detailed Framework Implementation Technical Requirements

### 3.1 GoFiber Implementation (Go)

Go utilizes Garbage Collector (GC) based memory management and Goroutines for concurrency. Monitoring these components is critical.

#### A. Required System & Memory Metrics

| Metric | Type | Description |
| :--- | :--- | :--- |
| `go_goroutines` | Gauge | Number of running goroutines (Goroutine Leak detection). |
| `go_memstats_sys_bytes` | Gauge | Total memory obtained from the OS. |
| `go_memstats_alloc_bytes` | Gauge | Memory allocated in the Heap for active objects. |
| `go_memstats_heap_idle_bytes` | Gauge | Memory on the heap that is unused but not yet returned to the OS. |
| `go_gc_duration_seconds` | Summary | Statistics on Garbage Collection pause durations. |

#### B. Technical Stack & Implementation

- **Library:** Native implementation using `prometheus/client_golang` wrapped with GoFiber's `adaptor`.
- **Security:** The `/metrics` endpoint is public and unprotected to simplify Prometheus Server scraping.

```go
package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	app := fiber.New()

	// Expose /metrics
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	app.Get("/user/:id", func(c fiber.Ctx) error {
		return c.SendString("User Profile")
	})

	app.Listen(":3000")
}
```

---

### 3.2 Hono Implementation (Bun Runtime)

Hono runtime runs exclusively on Bun in this architecture. The primary focus of telemetry here is to monitor the Event Loop health and JavaScriptCore (JSC) memory allocation.

#### A. Required System & Memory Metrics

| Metric | Type | Description |
| :--- | :--- | :--- |
| `process_resident_set_size_bytes` | Gauge | RSS — Total physical RAM used by the process. |
| `bun_edge_format_memory` | Gauge | Monitors Garbage Collection allocations specific to JavaScriptCore. |
| `eventloop_lag_seconds` | Gauge | Measures the delay of the event loop for synchronous blocking detection. |

#### B. Technical Stack & Implementation

- **Library:** `@hono/prometheus` (Official Prometheus middleware for Hono).

```ts
import { Hono } from 'hono'
import { prometheus } from '@hono/prometheus'

const app = new Hono()
const { printMetrics, registerMetrics } = prometheus()

// Apply metrics interceptor to all routes
app.use('*', registerMetrics)

// Expose the metrics endpoint
app.get('/metrics', printMetrics)

app.get('/products/:id', (c) => c.text(`Product ${c.req.param('id')}`))

export default app
```

---

### 3.3 ElysiaJS Implementation (Bun Runtime)

ElysiaJS runs exclusively on the Bun Runtime utilizing JavaScriptCore (JSC).

#### A. Required System & Memory Metrics

| Metric | Type | Description |
| :--- | :--- | :--- |
| `process_resident_set_size_bytes` | Gauge | Absolute physical RAM consumed by the Bun process. |
| `bun_edge_format_memory` | Gauge | Monitors Garbage Collection allocations specific to JavaScriptCore. |

#### B. Technical Stack & Implementation

- **Library:** `@elysiajs/prometheus` (Official Prometheus plugin for Elysia).

```ts
import { Elysia } from 'elysia'
import { prometheus } from '@elysiajs/prometheus'

const app = new Elysia()
  .use(prometheus()) // Automatically mounts to /metrics
  .get('/v1/item/:id', ({ params: { id } }) => `Item ${id}`)
  .listen(3001)
```

---

## 4. Operational & Alerting Requirements

### 4.1 Memory Critical Alert

| Field | Value |
| :--- | :--- |
| **Condition** | `process_resident_set_size_bytes` (or `go_memstats_sys_bytes`) > 85% of container/VPS limit for 3 minutes |
| **Action** | Send a critical alert to Slack/Discord/PagerDuty before an OOM Kill occurs |

### 4.2 API Failure Rate Alert

| Field | Value |
| :--- | :--- |
| **Condition** | Ratio of `http_requests_total{status=~"5.."}` ÷ total requests > 2% within 5 minutes |
| **Action** | "High Error Rate 5XX" warning on the related service |

---

## 5. Security & Performance Constraints

- **Scraping Security:** The `/metrics` endpoint may be internal-only. External access should be blocked via a reverse proxy or IP Whitelisting specifically for the Prometheus Server.
- **Allocation Overhead:** The middleware must consume a maximum additional CPU load of **2–3%** during peak traffic.
