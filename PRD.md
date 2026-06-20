# PRD.md — Go Imageboard API Wrapper

## Goal

Belajar Go dengan bikin sesuatu yang useful: REST API wrapper untuk Danbooru, Gelbooru, dan Rule34.
Project ini cover konsep Go yang paling sering dipakai di production backend.

## What You'll Learn

| Konsep Go | Dipelajari via |
|---|---|
| Structs & interfaces | Model normalisasi antar provider |
| Error handling idiom | Upstream API failures |
| HTTP client | Fetch ke Danbooru/Gelbooru/Rule34 |
| JSON marshal/unmarshal | Parse response provider |
| Context & timeout | HTTP client timeout |
| Dependency injection | Redis client sharing |
| Middleware | Swagger, logger, recover |
| Environment config | `.env` loading |

---

## Phases

### Phase 1 — Skeleton

**Goal**: Project bisa run, Swagger UI terbuka, satu endpoint dummy works.

Tasks:
- [ ] `go mod init`
- [ ] Install deps: `gofiber/fiber/v3`, `go-redis/v9`, `swaggo/swag`, `joho/godotenv`
- [ ] Setup `main.go` dengan Fiber app + basic Swagger route
- [ ] Setup `config/config.go` — load `.env`
- [ ] Verify `swag init` generates `docs/`
- [ ] Satu dummy endpoint `GET /ping` dengan Swagger annotation

**Done when**: `go run main.go` jalan, `/swagger/` terbuka di browser, `/ping` terdokumentasi.

---

### Phase 2 — Redis Cache Layer

**Goal**: Bisa set/get cache, understand TTL.

Tasks:
- [ ] Setup `cache/redis.go` — init client, `Get`, `Set`, `Delete` helpers
- [ ] Test manual via `redis-cli` atau Redisinsight
- [ ] Cache key convention: `provider:resource:tags_hash`

**Done when**: Set key dari Go, get dari redis-cli (atau sebaliknya).

---

### Phase 3 — Provider Bindings

**Goal**: Fetch real data dari ketiga provider, normalize ke shared struct.

Shared `models.Post`:
```go
type Post struct {
    ID        int      `json:"id"`
    Provider  string   `json:"provider"`
    FileURL   string   `json:"file_url"`
    PreviewURL string  `json:"preview_url"`
    Tags      []string `json:"tags"`
    Rating    string   `json:"rating"` // s, q, e
    Score     int      `json:"score"`
    Width     int      `json:"width"`
    Height    int      `json:"height"`
}
```

Tasks per provider:
- [ ] `providers/danbooru.go` — fetch + map ke `Post`
- [ ] `providers/gelbooru.go` — fetch + map ke `Post`
- [ ] `providers/rule34.go` — fetch + map ke `Post`

**Done when**: Bisa fetch posts dari semua 3 provider via Go code (unit test atau main).

---

### Phase 4 — Handlers + Routing

**Goal**: Expose provider bindings via HTTP endpoints dengan Swagger docs.

Endpoints:

```
GET /api/danbooru/posts?tags=&limit=&page=
GET /api/gelbooru/posts?tags=&limit=&page=
GET /api/rule34/posts?tags=&limit=&page=
GET /api/all/posts?tags=&limit=   ← aggregate semua provider
```

Tasks:
- [ ] `handlers/danbooru.go` — dengan full Swagger annotation
- [ ] `handlers/gelbooru.go`
- [ ] `handlers/rule34.go`
- [ ] `handlers/all.go` — concurrent fetch pakai goroutine + `sync.WaitGroup`
- [ ] `router/router.go` — register semua routes
- [ ] Wire Redis cache di setiap handler (check cache first, fetch if miss, store result)

**Done when**: Semua endpoint jalan + terdokumentasi di Swagger UI.

---

### Phase 5 — Polish

**Goal**: Production-ready habits.

Tasks:
- [ ] Global error handler di Fiber
- [ ] Request logger middleware
- [ ] Recover middleware (prevent crash on panic)
- [ ] Rate limit middleware (gofiber/contrib/limiter)
- [ ] Cache headers (`Cache-Control`) di response
- [ ] README.md — cara run, endpoint list

---

## Endpoint Contract

### `GET /api/{provider}/posts`

Query params:
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| tags | string | "" | Space-separated tags |
| limit | int | 20 | Max results (cap 100) |
| page | int | 1 | Page number |

Response `200`:
```json
{
  "provider": "danbooru",
  "count": 20,
  "posts": [
    {
      "id": 123,
      "provider": "danbooru",
      "file_url": "https://...",
      "preview_url": "https://...",
      "tags": ["1girl", "solo"],
      "rating": "s",
      "score": 42,
      "width": 1920,
      "height": 1080
    }
  ]
}
```

Response `502`:
```json
{
  "error": "upstream fetch failed",
  "provider": "danbooru"
}
```

---

## Dependencies

```bash
go get github.com/gofiber/fiber/v3
go get github.com/go-redis/redis/v9
go get github.com/swaggo/swag/cmd/swag
go get github.com/gofiber/contrib/swagger
go get github.com/joho/godotenv
```

## .env Template

```env
PORT=3000
REDIS_URL=redis://localhost:6379

# optional — untuk rate limit bypass di Danbooru & Gelbooru
DANBOORU_API_KEY=
DANBOORU_LOGIN=
GELBOORU_API_KEY=
GELBOORU_USER_ID=
```
