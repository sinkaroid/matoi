<div align="center">
<a href="https://sinkaroid.github.io/matoi/"><img width="500" src="resources/project/images/matoi-header.png" alt="matoi"></a>

<h4 align="center">Unified REST + GraphQL API gateway for booru imageboards</h4>
<p align="center">
<a href="https://github.com/sinkaroid/matoi/actions/workflows/playground.yml"><img src="https://github.com/sinkaroid/matoi/actions/workflows/playground.yml/badge.svg"></a>
<a href="https://qlty.sh/gh/sinkaroid/projects/matoi"><img src="https://qlty.sh/gh/sinkaroid/projects/matoi/maintainability.svg" alt="Maintainability" /></a>
</p>

<a href="https://github.com/sinkaroid/matoi/blob/master/CONTRIBUTING.md">Contributing</a> •
<a href="https://sinkaroid.github.io/matoi">Playground</a> •
<a href="https://github.com/sinkaroid/matoi/issues/new/choose">Report Issues</a>

</div>

---

<a href="http://localhost:3000"><img align="right" src="resources/project/images/sinkaroid-matoi.png" width="390"></a>

- [sinkaroid/matoi](#)
  - [The problem](#the-problem)
  - [The solution](#the-solution)
  - [Features](#features)
  - [Supported providers](#supported-providers)
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
  - [Pronunciation](#pronunciation)
  - [Legal](#legal)
  - [Microservices](#microservices)

## The problem

Many developers consume booru imageboards as a source of data when building applications. However, each imageboard has its own API format, authentication requirements, and response structure — Danbooru speaks JSON, Gelbooru speaks XML, Rule34 has its own quirks, and none of them agree on field names.

Developers end up writing adapter after adapter, maintaining boilerplate for every booru variant. It's exhausting, error-prone, and a massive time sink.

## The solution

Matoi provides a unified gateway that normalizes every provider response into a single `models.Post` struct with identical JSON outputs. Write your integration once. Change the provider name in the URL. Get the same schema back.

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

## Features

- **Unified JSON** — All providers return identical response structure. Provider name is the only variable.
- **REST + GraphQL** — Dual-protocol with identical feature parity. REST for simplicity, GraphQL for precision.
- **15 providers** — From Rule34 to Derpibooru. SFW and NSFW. Broadest booru coverage available.
- **Redis caching** — Response caching via Redis with configurable TTL. Redis is **required** for caching to function.
- **Media proxy** — Built-in reverse proxy bypasses hotlink protection. Returns local `matoi_file_url`, `matoi_preview_url`, `matoi_sample_url`.
- **Query completion** — [eiyuu](https://www.npmjs.com/package/eiyuu) improved autocomplete via goquery web scraping. Works where official APIs fall short.
- **Prometheus metrics** — Native `/metrics` endpoint with HTTP request duration, status code counters, and active connections.
- **Shuffle support** — Randomize result order with `?shuffle=1` or `?random=true`. Works across all 15 providers.
- **API key auth** — Simple Bearer token or query parameter authentication for protected routes.
- **FlareSolverr** — Optional Cloudflare bypass for providers behind WAF using a custom memory-monitored Docker image.
- **Containerized** — Multi-stage Alpine Docker build with automated GHCR publishing on version bump.
- **Swagger docs** — Interactive API documentation at `/swagger` with auto-generated OpenAPI spec.
- **GraphiQL playground** — Interactive GraphQL IDE at `/graphql` for query exploration.

## Supported providers

| #   | Provider         | Base endpoint        | Auth         | Query Completion |
| --- | ---------------- | -------------------- | ------------ | ---------------- |
| 1   | **Rule34**       | `rule34.xxx`         | Optional     | Eiyuu adapts     |
| 2   | **Danbooru**     | `danbooru.donmai.us` | Optional     | `tags.json` API  |
| 3   | **Gelbooru**     | `gelbooru.com`       | Optional     | Eiyuu adapts     |
| 4   | **Tbib**         | `tbib.org`           | Optional     | Eiyuu adapts     |
| 5   | **Xbooru**       | `xbooru.com`         | Optional     | Eiyuu adapts     |
| 6   | **Hypnohub**     | `hypnohub.net`       | Optional     | Eiyuu adapts     |
| 7   | **Safebooru**    | `safebooru.org`      | Optional     | Eiyuu adapts     |
| 8   | **Yande.re**     | `yande.re`           | Optional     | Eiyuu adapts     |
| 9   | **Konachan.com** | `konachan.com`       | Optional     | Eiyuu adapts     |
| 10  | **Konachan.net** | `konachan.net`       | Optional     | Eiyuu adapts     |
| 11  | **E621**         | `e621.net`           | **Required** | `tags.json` API  |
| 12  | **E926**         | `e926.net`           | **Required** | `tags.json` API  |
| 13  | **Furbooru**     | `furbooru.org`       | **Required** | `tags.json` API  |
| 14  | **Derpibooru**   | `derpibooru.org`     | **Required** | `tags.json` API  |
| 15  | **Realbooru**    | `realbooru.com`      | Optional     | Eiyuu adapts     |

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
- All layers communicate via dependency injection — no globals, no `init()` ordering issues

## Prerequisites

> **Redis is required** for caching to function. Without a Redis connection, all requests will fall through to upstream on every call — no caching, no deduplication, and full upstream rate limit exposure. For self-hosted production use, a running Redis instance is expected.

- Go 1.24+
- Redis
  - You can use `docker pull ghcr.io/sinkaroid/matoi-redis:latest` for full control.
  - If just small usage or experimenting, You can get [redis.io/try-free](https://redis.io/try-free/) for demo and free tier available.

### Docker

Adjust port and your env file first.

```bash
docker pull ghcr.io/sinkaroid/matoi:latest
docker run -p 3000:3000 --env-file .env ghcr.io/sinkaroid/matoi:latest
```

Or build locally:

```bash
docker build -t matoi .
docker run -p 3000:3000 --env-file .env matoi
```

### Manual

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

### Configuration

All configuration is via environment variables or a `.env` file. See [`.env.schema`](./.env.schema) for the full template.

| Variable                               | Default                  | Required | Description                                |
| -------------------------------------- | ------------------------ | -------- | ------------------------------------------ |
| `MATOI_PORT`                           | `3000`                   | No       | Server port                                |
| `MATOI_GRAPHQL`                        | `false`                  | No       | Enable GraphQL endpoint                    |
| `MATOI_REDIS_URL`                      | `redis://localhost:6379` | **Yes**  | Redis connection string                    |
| `MATOI_REDIS_EXPIRE_CACHE`             | `5`                      | No       | Cache TTL in minutes                       |
| `MATOI_API_KEY`                        | `""`                     | No       | API key for protected routes               |
| `MATOI_USER_AGENT`                     | `matoi/{version}`        | No       | Upstream User-Agent                        |
| `MATOI_RESOLVER_URL`                   | `""`                     | No       | External base URL for media proxy          |
| `MATOI_ENABLE_LOGS`                    | `false`                  | No       | Enable request logging                     |
| `MATOI_POST_REST_RETURN_LIMIT`         | `100`                    | No       | Max posts per page                         |
| `MATOI_FLARESOLVERR_URL`               | `""`                     | No       | FlareSolverr endpoint                      |
| `DANBOORU_API_ID` / `DANBOORU_API_KEY` | `""`                     | No       | Danbooru credentials                       |
| `GELBOORU_API_ID` / `GELBOORU_API_KEY` | `""`                     | **Yes**  | Gelbooru credentials                       |
| `RULE34_API_ID` / `RULE34_API_KEY`     | `""`                     | No       | Rule34 credentials                         |
| `E621_API_ID` / `E621_API_KEY`         | `""`                     | **Yes**  | E621 credentials (provider required)       |
| `E926_API_ID` / `E926_API_KEY`         | `""`                     | **Yes**  | E926 credentials (provider required)       |
| `FURBOORU_API_KEY`                     | `""`                     | **Yes**  | Furbooru credentials (provider required)   |
| `DERPIBOORU_API_KEY`                   | `""`                     | **Yes**  | Derpibooru credentials (provider required) |

## API reference

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
GET /api/{provider}/query_completion?tags=jeanne
```

> **No caching.** Autocomplete results are always fetched live to prevent stale or empty responses.

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

Proxies image and media content, bypassing upstream hotlink protection. Returns raw binary with correct `Content-Type` headers. All `matoi_*_url` fields in post responses resolve through this endpoint. Media proxy routes are publicly accessible — no authentication required.

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

The GraphQL schema maintains 100% feature parity with REST — including caching, query completion, media proxy URLs, and pagination. Interactive GraphiQL playground is available at `GET /graphql` (publicly accessible).

See [graphql.md](./docs/graphql.md) for detailed GraphQL documentation with cURL, Node.js, Bun, and Go examples.

### Response format

Every endpoint returns a consistent normalized envelope.

**Success:**

```json
{
  "success": true,
  "provider": "danbooru",
  "count": 100,
  "posts": []
}
```

**Empty / no results:**

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

All errors return a consistent structured response:

```json
{
  "success": false,
  "reason": "Invalid or missing API key"
}
```

| Status | Meaning                    |
| ------ | -------------------------- |
| `200`  | Success                    |
| `401`  | Invalid or missing API key |
| `404`  | No results found           |
| `502`  | Upstream API fetch failed  |

## Authentication

Protected endpoints (`/api/*` routes) require an API key. Pass via query parameter or Authorization header.

**Query parameter:**

```
GET /api/danbooru/posts?api_key=YOUR_KEY&tags=yuri
```

**Authorization header:**

```
Authorization: Bearer YOUR_KEY
```

Media proxy endpoints (`/api/{provider}/media`) and system endpoints (`/`, `/ping`, `/metrics`, `/swagger`, `/graphql`) are publicly accessible without authentication.

## Caching

Matoi uses Redis for response caching. **Redis is required** — without it, every request hits upstream directly with no deduplication.

| Property         | Detail                                                                 |
| ---------------- | ---------------------------------------------------------------------- |
| **Key format**   | `provider:resource:param` — e.g. `danbooru:posts:yuri_1girl_page=1`    |
| **Default TTL**  | 5 minutes (configurable via `MATOI_REDIS_EXPIRE_CACHE`)                |
| **Cache format** | Serialized JSON of the parsed response struct (not raw HTTP bytes)     |
| **Cache miss**   | Fetch from upstream → serialize → store in Redis → return to client    |
| **Degradation**  | If Redis is unreachable, requests proceed uncached — no crash, no hang |
| **Completion**   | Query completion is **never cached** to prevent stale autocomplete     |

## Prometheus metrics

Matoi exposes native Prometheus metrics at `GET /metrics` (publicly accessible, no auth required):

| Metric                          | Type      | Description                                          |
| ------------------------------- | --------- | ---------------------------------------------------- |
| `http_requests_total`           | Counter   | Total HTTP requests by status code, method, and path |
| `http_request_duration_seconds` | Histogram | Request latency distribution                         |
| `http_requests_active`          | Gauge     | Currently active in-flight requests                  |

Metrics are registered via a custom Fiber middleware in [`middleware/prometheus.go`](./middleware/prometheus.go).

## Rate limiting

Matoi does not enforce its own rate limits. Rate limiting is handled upstream by each provider's API constraints. Matoi passes through provider-specific `Retry-After` headers when available.

To reduce upstream rate limit exposure:

- **Enable Redis caching** — repeated identical requests are served from cache rather than hitting upstream
- **Provide API credentials** — authenticated requests generally receive higher upstream rate limits
- **Use FlareSolverr** — bypasses Cloudflare-based throttling on affected providers

## FlareSolverr

Some providers (notably Konachan.com) sit behind Cloudflare WAF. Matoi supports optional FlareSolverr integration to bypass this.

```env
MATOI_FLARESOLVERR_URL=http://localhost:8191/v1
```

A custom Docker image with memory monitoring is provided:

```bash
docker pull ghcr.io/sinkaroid/matoi-flaresolverr:latest
```

The custom image bundles a Go memory monitoring sidecar (`memapi`) that reads the Linux `/proc` filesystem to calculate exact RSS of the FlareSolverr process tree — more accurate than cgroup-based metrics. See [`flaresolverr/`](./flaresolverr) for implementation details.

## Eiyuu query completion

Providers without a native tag completion API use an Eiyuu-style web scraping approach, ported from the [eiyuu](https://npmjs.com/package/eiyuu) npm module:

1. Sends an HTTP request to the provider's tag suggestion endpoint
2. Parses the HTML response using `goquery` (Go's equivalent of Cheerio/jQuery)
3. Extracts tag names, sanitizes HTML entities via `html.UnescapeString`, and URL-decodes via `url.QueryUnescape`
4. Returns a clean JSON array of tag strings

Results are double-decoded to prevent HTML entity leakage in JSON output. Eiyuu results are never cached — always fetched live.

## Development

This project uses [go-task/task](https://github.com/go-task/task) as task runner. Please check available commands via `task`.

Each integration test verifies:

1. Posts endpoint returns valid data with correct structure
2. Pagination returns different results across pages
3. Media proxy serves binary content with correct headers
4. Query completion returns non-empty tag suggestions
5. GraphQL response matches REST endpoint parity

### Swagger docs

Handler annotations follow standard `swag` format and regenerate after any handler change `swag init`

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

## Pronunciation

[`ja_JP`](https://www.localeplanet.com/java/ja-JP/index.html) • **/ma·to·i/** — **纏** (_matoi_) **"to gather"**, **"to collect"**, or **"to bring together into one"**. The logo and header is inspired by Senketsu (Scissor Blade) & Ryūko Matoi from Kill la Kill.

## Legal

This tool can be freely copied, modified, altered, distributed without any attribution whatsoever. However, if you feel
like this tool deserves an attribution, mention it. It won't hurt anybody.

> Licence: WTF.

## Microservices

Microservices and subprojects is part of a broader ecosystem of specialized services, each focused on a specific platform or content source while sharing a common design philosophy maintained by [ScathachGrip](https://github.com/ScathachGrip)

- **sinkaroid/matoi — Unified REST and GraphQL gateway for booru-based imageboards.**
- [sinkaroid/jandapress](https://github.com/sinkaroid/jandapress) — Unified REST and GraphQL API for nhentai and other doujinshi
- [sinkaroid/pixivHono](https://github.com/sinkaroid/pixivHono) — Unified REST and GraphQL API for Pixiv
- [sinkaroid/lustpress](https://github.com/sinkaroid/lustpress) — Unified REST and GraphQL API for PornHub and other R18 platforms

Each service is developed independently, enabling modular deployments, isolated maintenance, and platform-specific optimizations while remaining interoperable within the ecosystem.
