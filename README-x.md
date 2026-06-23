# sinkaroid/matoi

Unified REST + GraphQL API gateway for booru imageboards

<a href="https://github.com/sinkaroid/matoi"><img align="right" src="resources/project/images/sinkaroid-matoi.png" width="350"></a>

- [sinkaroid/matoi](#)
  - [The problem](#the-problem)
  - [The solution](#the-solution)
  - [Supported providers](#supported-providers)
  - [Features](#features)
  - [Architecture](#architecture)
  - [Prerequisites + Installation](#prerequisites)
    - [Docker](#docker)
    - [Manual](#manual)
    - [Configuration](#configuration)
  - [API reference](#api-reference)
    - [Posts](#posts)
    - [Query completion](#query-completion)
    - [Media proxy](#media-proxy)
    - [GraphQL](#graphql)
    - [Response format](#response-format)
    - [Error handling](#error-handling)
  - [Authentication](#authentication)
  - [Caching](#caching)
  - [Prometheus metrics](#prometheus-metrics)
  - [Rate limiting](#rate-limiting)
  - [FlareSolverr](#flaresolverr)

---

<div align="center">

<a href="https://github.com/sinkaroid/matoi/actions/workflows/playground.yml"><img src="https://github.com/sinkaroid/matoi/actions/workflows/playground.yml/badge.svg"></a>
<a href="https://qlty.sh/gh/sinkaroid/projects/matoi"><img src="https://qlty.sh/gh/sinkaroid/projects/matoi/maintainability.svg" alt="Maintainability" /></a>

Unified REST + GraphQL API gateway for booru imageboards — normalizes 15+ imageboard APIs into a single consistent interface.
Swap the provider name in your URL, get the same JSON schema back.

<b>
<a href="https://github.com/sinkaroid/matoi/blob/master/CONTRIBUTING.md">Contributing</a> •
<a href="https://sinkaroid.github.io/matoi">Playground</a> •
<a href="https://github.com/sinkaroid/matoi/issues/new/choose">Report Issues</a>
</b>
</div>

## The problem

Many developers consume booru imageboards as a source of data when building applications. However, each imageboard has its own API format, authentication requirements, and response structure — Danbooru speaks JSON, Gelbooru speaks XML, Rule34 has its own quirks, and none of them agree on field names.

Developers end up writing adapter after adapter, maintaining boilerplate for every booru variant. It's exhausting, error-prone, and a massive time sink.

## The solution

Matoi provides a unified gateway that normalizes every provider response into a single `models.Post` struct with identical JSON output:

```json
{
  "success": true,
  "provider": "...",
  "count": 100,
  "posts": [
    {
      "id": "",
      "directory": "...",
      "file_url": "...",
      "preview_url": "...",
      "sample_url": "...",
      "matoi_file_url": "...",
      "matoi_preview_url": "...",
      "matoi_sample_url": "...",
      "rating": "...",
      "score": 0,
      "source": "...",
      "image": "...",
      "tags": [],
      "link": "..."
    }
  ]
}
```

Write your integration once. Change the provider name in the URL. Get the same schema back.

## Supported providers

| #   | Provider         | Base endpoint        | Auth         | Completion method |
| --- | ---------------- | -------------------- | ------------ | ----------------- |
| 1   | **Rule34**       | `api.rule34.xxx`     | Optional     | Eiyuu scraping    |
| 2   | **Danbooru**     | `danbooru.donmai.us` | Optional     | `tags.json` API   |
| 3   | **Gelbooru**     | `gelbooru.com`       | Optional     | Eiyuu scraping    |
| 4   | **Tbib**         | `tbib.org`           | Optional     | Eiyuu scraping    |
| 5   | **Xbooru**       | `xbooru.com`         | Optional     | Eiyuu scraping    |
| 6   | **Hypnohub**     | `hypnohub.net`       | Optional     | Eiyuu scraping    |
| 7   | **Safebooru**    | `safebooru.org`      | Optional     | Eiyuu scraping    |
| 8   | **Yande.re**     | `yande.re`           | Optional     | Eiyuu scraping    |
| 9   | **Konachan.com** | `konachan.com`       | Optional     | Eiyuu scraping    |
| 10  | **Konachan.net** | `konachan.net`       | Optional     | Eiyuu scraping    |
| 11  | **E621**         | `e621.net`           | **Required** | `tags.json` API   |
| 12  | **E926**         | `e926.net`           | **Required** | `tags.json` API   |
| 13  | **Furbooru**     | `furbooru.org`       | **Required** | `tags.json` API   |
| 14  | **Derpibooru**   | `derpibooru.org`     | **Required** | `tags.json` API   |
| 15  | **Realbooru**    | `realbooru.com`      | Optional     | Eiyuu scraping    |

## Features

- **Unified JSON** — All providers return identical response structure. Provider name is the only variable.
- **REST + GraphQL** — Dual-protocol with identical feature parity. REST for simplicity, GraphQL for precision.
- **15 providers** — From Rule34 to Derpibooru. SFW and NSFW. Broadest booru coverage available.
- **Redis caching** — Configurable TTL caching via Redis with graceful fallback when unavailable.
- **Media proxy** — Built-in reverse proxy bypasses hotlink protection. Returns local `matoi_file_url`, `matoi_preview_url`, `matoi_sample_url`.
- **Query completion** — Eiyuu-style tag autocomplete via goquery web scraping. Works where official APIs fall short.
- **Prometheus metrics** — Native `/metrics` endpoint with HTTP request duration, status code counters, and active connections.
- **Shuffle support** — Randomize result order with `?shuffle=1`. Works across all 15 providers.
- **API key auth** — Simple Bearer token or query parameter authentication for protected routes.
- **FlareSolverr** — Optional Cloudflare bypass for blocked providers using a custom memory-monitored Docker image.
- **Containerized** — Multi-stage Alpine Docker build with automated GHCR publishing on version bump.
- **CI/CD** — Automated linting, Docker builds, GitHub Pages Swagger UI deployment, and release pipelines.
- **Swagger docs** — Interactive API documentation at `/swagger` with auto-generated OpenAPI spec.
- **GraphiQL playground** — Interactive GraphQL IDE at `/graphql` for query exploration.

## Project structure

```
.
├── main.go                 # Entry point, provider/handler wiring
├── config/
│   └── config.go           # Environment loading & app version
├── cache/
│   └── redis.go            # Redis client with Get/Set/Delete helpers
├── models/
│   └── post.go             # Normalized Post struct (single source of truth)
├── providers/              # 15 upstream API integrations
│   ├── danbooru.go
│   ├── gelbooru.go
│   ├── rule34.go
│   ├── tbib.go
│   ├── xbooru.go
│   ├── hypnohub.go
│   ├── safebooru.go
│   ├── yandere.go
│   ├── konachan_com.go
│   ├── konachan_net.go
│   ├── e621.go
│   ├── e926.go
│   ├── furbooru.go
│   ├── derpibooru.go
│   └── realbooru.go
├── handlers/               # HTTP endpoint handlers
│   ├── home.go             # Healthcheck / system info
│   ├── graphql.go          # GraphQL schema & resolvers (code-first)
│   ├── danbooru.go
│   ├── gelbooru.go
│   └── ...
├── router/
│   └── router.go           # Route registration & middleware setup
├── middleware/
│   ├── auth.go             # API key authentication (query + bearer)
│   └── prometheus.go       # Prometheus HTTP metrics middleware
├── docs/                   # Auto-generated Swagger docs (swag init)
├── flaresolverr/           # Custom FlareSolverr memory monitoring sidecar
├── dev_tools/              # Integration tests & docs generator
├── Dockerfile              # Multi-stage Alpine build
├── Taskfile.yml            # Task runner (dev, lint, build, test, docs)
└── .github/workflows/      # CI/CD pipelines (dockerized, lint, release, playground)
```

## Architecture

Matoi follows a layered architecture with dependency injection:

```
Client → Fiber Router → Auth Middleware → Handler → Provider → Upstream API
                          ↓                              ↓
                     Redis Cache                   goquery Scraper
                          ↓                              ↓
                     Response ←————— Normalize to models.Post ——————
```

- **Router layer** — Registers routes, wires middleware (CORS, auth, logging, Prometheus)
- **Handler layer** — HTTP logic, request parsing, JSON serialization, `resolveMatoiURLs()`
- **Provider layer** — Upstream fetch, raw struct mapping, XML/JSON parsing
- **Cache layer** — Redis read-through cache with configurable TTL
- **All layers** communicate via dependency injection — no globals, no init() ordering issues

## Prerequisites

<table><td><b>Note:</b> Redis is optional. Matoi falls back gracefully if Redis is unavailable — requests will fetch directly from upstream without caching.</td></table>

- Go 1.26.4+
- Redis (optional)
- Task CLI (optional, for dev commands)

### Installation

#### Docker

```bash
docker pull ghcr.io/sinkaroid/matoi:latest
docker run -p 3000:3000 --env-file .env ghcr.io/sinkaroid/matoi:latest
```

Or build locally:

```bash
docker build -t matoi .
docker run -p 3000:3000 --env-file .env matoi
```

#### Manual

```bash
git clone https://github.com/sinkaroid/matoi.git
cd matoi
go mod tidy
go run main.go
```

With hot-reload (requires [air](https://github.com/air-verse/air)):

```bash
task dev
```

#### Configuration

All configuration via environment variables or `.env` file. See [`.env.schema`](./.env.schema) for the full template.

| Variable                       | Default                  | Description                       |
| ------------------------------ | ------------------------ | --------------------------------- |
| `MATOI_PORT`                   | `3000`                   | Server port                       |
| `MATOI_GRAPHQL`                | `false`                  | Enable GraphQL                    |
| `MATOI_REDIS_URL`              | `redis://localhost:6379` | Redis connection string           |
| `MATOI_REDIS_EXPIRE_CACHE`     | `5`                      | Cache TTL in minutes              |
| `MATOI_API_KEY`                | `""`                     | API key for protected routes      |
| `MATOI_USER_AGENT`             | `matoi/{version}`        | Upstream User-Agent               |
| `MATOI_RESOLVER_URL`           | `""`                     | External base URL for media proxy |
| `MATOI_ENABLE_LOGS`            | `false`                  | Request logging                   |
| `MATOI_POST_REST_RETURN_LIMIT` | `100`                    | Max posts per page                |
| `MATOI_FLARESOLVERR_URL`       | `""`                     | FlareSolverr endpoint             |
| `DANBOORU_API_ID/KEY`          | `""`                     | Danbooru credentials              |
| `GELBOORU_API_ID/KEY`          | `""`                     | Gelbooru credentials              |
| `RULE34_API_ID/KEY`            | `""`                     | Rule34 credentials                |
| `E621_API_ID/KEY`              | `""`                     | E621 credentials                  |
| `E926_API_ID/KEY`              | `""`                     | E926 credentials                  |
| `FURBOORU_API_KEY`             | `""`                     | Furbooru credentials              |
| `DERPIBOORU_API_KEY`           | `""`                     | Derpibooru credentials            |

## API reference

### Health & system

| Method | Endpoint   | Description                                       | Auth |
| ------ | ---------- | ------------------------------------------------- | ---- |
| GET    | `/`        | System info (version, uptime, memory, server geo) | ❌   |
| GET    | `/ping`    | Liveness check                                    | ❌   |
| GET    | `/metrics` | Prometheus metrics                                | ❌   |
| GET    | `/swagger` | Swagger UI documentation                          | ❌   |
| GET    | `/graphql` | GraphiQL interactive playground                   | ❌   |

### Posts

```
GET /api/{provider}/posts?tags=&limit=&page=&shuffle=0
```

| Param     | Type   | Default | Description                   |
| --------- | ------ | ------- | ----------------------------- |
| `tags`    | string | `""`    | Space-separated tag filter    |
| `limit`   | int    | `100`   | Results per page              |
| `page`    | int    | `1`     | Page number                   |
| `shuffle` | int    | `0`     | Set to `1` to randomize order |

**Response 200:**

```json
{
  "success": true,
  "provider": "danbooru",
  "count": 100,
  "posts": [
    {
      "id": 1234567,
      "directory": "sample",
      "file_url": "https://danbooru.donmai.us/...",
      "preview_url": "https://danbooru.donmai.us/...",
      "sample_url": "https://danbooru.donmai.us/...",
      "matoi_file_url": "http://localhost:3000/api/danbooru/media?url=...",
      "matoi_preview_url": "http://localhost:3000/api/danbooru/media?url=...",
      "matoi_sample_url": "http://localhost:3000/api/danbooru/media?url=...",
      "rating": "s",
      "score": 42,
      "source": "https://...",
      "image": "filename.jpg",
      "tags": ["1girl", "solo", "cat_ears"],
      "link": "https://danbooru.donmai.us/posts/1234567"
    }
  ]
}
```

**Response 404 (no results):**

```json
{
  "success": false,
  "provider": "danbooru",
  "count": 0,
  "posts": []
}
```

### Query completion

```
GET /api/{provider}/query_completion?tags=
```

> **No caching.** Autocomplete results are fetched live on every request to prevent stale or empty responses.

```json
{
  "success": true,
  "tags": ["jeanne_d'arc_(fate)", "jeanne_(fate)", "jeanne_(fate/apocrypha)"]
}
```

### Media proxy

```
GET /api/{provider}/media?url=
```

Proxies image/media content bypassing hotlink protection. Returns raw binary with proper `Content-Type` headers. All `matoi_*_url` fields point here. Media proxy endpoints are publicly accessible (no auth required).

### GraphQL

Matoi exposes a unified GraphQL endpoint at `POST /api/graphql`. Enable via `.env`:

```env
MATOI_GRAPHQL=true
```

Query all providers in a single request:

```graphql
query {
  danbooru {
    posts(tags: "yuri", limit: 5) {
      id
      file_url
      matoi_file_url
      rating
      tags
    }
    completion(tags: "yuri")
  }
}
```

The GraphQL schema maintains 100% feature parity with REST — including caching, query completion, media proxy URLs, and pagination. Interactive GraphiQL playground at `GET /graphql` (unprotected).

See [USAGE.md](./USAGE.md) for detailed GraphQL documentation with cURL, Node.js, Bun, and Go examples.

### Response format

Every endpoint follows a strict normalized envelope:

**Success:**

```json
{
  "success": true,
  "provider": "danbooru",
  "count": 100,
  "posts": []
}
```

**Empty / No results:**

```json
{
  "success": false,
  "provider": "danbooru",
  "count": 0,
  "posts": []
}
```

> Empty results return `HTTP 404` — never `200`. Empty arrays are always `[]`, never `null`.

### Error handling

All errors return a consistent fiber.Error response:

```json
{
  "success": false,
  "reason": "Invalid or missing API key"
}
```

HTTP status codes:
| Code | Meaning |
|------|---------|
| 200 | Success |
| 401 | Invalid/missing API key |
| 404 | No results found |
| 502 | Upstream API fetch failed |

## Authentication

Protected endpoints (`/api/*` routes) require an API key. Pass via:

**Query parameter:**

```
GET /api/danbooru/posts?api_key=YOUR_KEY&tags=yuri
```

**Authorization header:**

```
Authorization: Bearer YOUR_KEY
```

Media proxy endpoints (`/api/{provider}/media`) and health endpoints (`/`, `/ping`, `/metrics`, `/swagger`, `/graphql`) are publicly accessible.

## Caching

Matoi uses Redis for response caching with the following strategy:

- **Cache key format**: `provider:resource:param` (e.g. `danbooru:posts:yuri_1girl_page=1`)
- **Default TTL**: 5 minutes (configurable via `MATOI_REDIS_EXPIRE_CACHE`)
- **Cache format**: Serialized JSON of the parsed response struct (not raw HTTP)
- **Cache miss**: Fetch from upstream, serialize, store in Redis, return
- **Graceful degradation**: If Redis is unavailable, requests proceed without caching — no crash, no hang
- **Query completion**: Explicitly **not cached** to prevent stale autocomplete results

## Prometheus metrics

Matoi exposes native Prometheus metrics at `GET /metrics` (unprotected):

| Metric                          | Type      | Description                                          |
| ------------------------------- | --------- | ---------------------------------------------------- |
| `http_requests_total`           | Counter   | Total HTTP requests by status code, method, and path |
| `http_request_duration_seconds` | Histogram | Request latency distribution                         |
| `http_requests_active`          | Gauge     | Currently active requests                            |

Metrics are registered via a custom Fiber middleware in [`middleware/prometheus.go`](./middleware/prometheus.go).

## Rate limiting

Rate limiting is handled upstream by each provider's API constraints. Matoi passes through provider-specific limits and retry-after headers when available. Using a FlareSolverr instance or providing API credentials generally increases rate limits for most providers.

## FlareSolverr

Some providers (notably Konachan.com) are behind Cloudflare protection. Matoi supports optional FlareSolverr bypass.

```env
MATOI_FLARESOLVERR_URL=http://localhost:8191/v1
```

A custom Docker image with memory monitoring is available:

```bash
docker pull ghcr.io/sinkaroid/matoi-flaresolverr:latest
```

The custom image includes a Go memory monitoring sidecar (`memapi`) that reads the Linux `/proc` filesystem to calculate exact RSS of the FlareSolverr process tree — more accurate than cgroup metrics. See [`flaresolverr/`](./flaresolverr) for details.

## Eiyuu query completion

Providers that lack a native completion API use Eiyuu — a web scraping approach ported from the [eiyuu](https://npmjs.com/package/eiyuu) npm module:

1. Sends an HTTP request to the provider's tag suggestion endpoint
2. Parses the HTML response using `goquery` (Go's equivalent of Cheerio/jQuery)
3. Extracts tag names, sanitizes HTML entities (`html.UnescapeString`), and URL-decodes (`url.QueryUnescape`)
4. Returns a clean JSON array of tag strings

Results are double-decoded to prevent HTML entity leakage in JSON responses. Not cached — always fetches fresh data.

## Design philosophy

JSON output normalization is the project's primary objective. Every provider response is strictly mapped to the same `models.Post` struct, wrapped in an identical envelope:

```json
{ "success": true, "provider": "...", "count": N, "posts": [...] }
```

This means: **swap the provider name in your URL, get the same JSON schema back.** No more writing adapters for every booru variant. No more guessing field names. One integration, 15 providers.

The project is also a Go learning exercise covering:

- Structs, interfaces, and type composition
- Idiomatic error handling with `fiber.NewError`
- HTTP clients, context, and timeouts
- JSON/XML parsing and normalization
- Web scraping with goquery (DOM traversal)
- Redis caching and dependency injection
- Middleware chains and request lifecycle
- Prometheus instrumentation
- GraphQL schema design with code-first approach
- Multi-stage Docker builds and CI/CD

## Development

### Commands

```bash
task dev              # Hot-reload dev server (air)
task lint             # golangci-lint
task build            # Build production binary to bin/matoi.exe
task version          # Show application version
swag init             # Generate Swagger docs from annotations
task docs:generate    # Generate GH Pages Swagger UI
task flush-cache      # Flush Redis cache
task kill-dev         # Kill zombie processes on port 3000
```

### Testing

```bash
# Integration tests (requires server running on :3000)
task test-matoi

# Individual provider tests
task test-danbooru
task test-e621
task test-e926
task test-furbooru
task test-derpibooru
task test-realbooru

# Goquery prototype test
task test-goquery
```

All integration tests verify:

1. ✅ Posts endpoint returns valid data with correct structure
2. ✅ Pagination returns different results across pages
3. ✅ Media proxy serves binary content correctly
4. ✅ Query completion returns tag suggestions
5. ✅ GraphQL parity matches REST endpoints

### Swagger docs

Handler annotations use standard `swag` format:

```go
// @Summary      Get posts from Danbooru
// @Tags         danbooru
// @Produce      json
// @Param        tags   query  string  false  "Space-separated tags"
// @Param        limit  query  int     false  "Results per page"
// @Param        page   query  int     false  "Page number"
// @Success      200  {object}  PostsResponse
// @Failure      401  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /api/danbooru/posts [get]
```

Regenerate after any handler change:

```bash
swag init
```

## CI/CD

| Pipeline       | Trigger                              | Description                                  |
| -------------- | ------------------------------------ | -------------------------------------------- |
| **Dockerized** | Push to `master` (with version bump) | Builds & pushes to `ghcr.io/sinkaroid/matoi` |
| **Lint**       | PR / Push                            | Runs `golangci-lint` full suite              |
| **Release**    | Tag push (`v*`)                      | Auto-release via goreleaser                  |
| **Playground** | Push to `master`                     | Deploys Swagger UI to GitHub Pages           |

The Docker pipeline also builds and pushes a custom FlareSolverr image (`ghcr.io/sinkaroid/matoi-flaresolverr`) when `flaresolverr/` changes.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

Please ensure:

- All code compiles (`go build`)
- Linter passes (`task lint`)
- Integration tests pass (`task test-matoi`)
- Swagger docs are regenerated after handler changes

## Legal

[MIT](./LICENSE) © sinkaroid
