# go-boilerplate

A Go backend boilerplate meant to be cloned for new services. Priorities: **security**, **performance**, **KISS**.

Stack: Echo v5 · pgx (pgxpool) · Redis (go-redis) · OpenTelemetry · Prometheus · slog.

## Installation

Requires Go 1.26+, Postgres, and Redis.

```bash
# 1. Clone
git clone <repo-url> my-service && cd my-service

# 2. Rename the module path for the new service
./rename.sh github.com/julles/my-service

# 3. Set up the environment
cp .env.example .env      # then fill in DATABASE_URL, REDIS_URL, etc.

# 4. Fetch dependencies
go mod tidy

# 5. Run
go run ./cmd/api        # HTTP server
go run ./cmd/worker     # queue consumer (background jobs)
go run ./cmd/scheduler  # cron (run a SINGLE instance)
```

Or bring up the whole stack at once (api + worker + scheduler + postgres + redis):

```bash
docker compose up --build
```

Configuration is injected at runtime (12-factor); `.env` is only for local tooling (docker-compose `env_file`, direnv). See `.env.example` for the full list of variables.

## Environment

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | ✅ | — | Postgres connection string |
| `REDIS_URL` | ✅ | — | Redis connection (rate limiter + cache + queue) |
| `HTTP_PORT` | | `:8080` | Listen address |
| `SERVER_NAME` | | `go-boilerplate` | Service name (tracing/logs) |
| `OTLP_ENDPOINT` | | empty | OTLP endpoint; empty disables tracing |
| `RATE_LIMIT` | | `100` | Requests per window |
| `RATE_WINDOW` | | `1m` | Window length |
| `WORKER_CONCURRENCY` | | `10` | Parallel tasks in the worker |
| `DB_MAX_CONNS` | | `10` | Max Postgres pool connections |
| `DB_MIN_CONNS` | | `2` | Min idle connections (warm) |
| `DB_MAX_CONN_LIFETIME` | | `1h` | Max connection lifetime |
| `DB_MAX_CONN_IDLE_TIME` | | `30m` | Idle time before a connection is closed |
| `REDIS_POOL_SIZE` | | `10` | Redis pool size |
| `REDIS_MIN_IDLE_CONNS` | | `0` | Min idle Redis connections |

### Per-binary pool tuning

`api`, `worker`, and `scheduler` are separate processes, so each can use a different pool size via its own environment — no code changes required:

```
api:        DB_MAX_CONNS=20   # high traffic
worker:     DB_MAX_CONNS=10   # ≈ WORKER_CONCURRENCY
scheduler:  DB_MAX_CONNS=2    # occasional cron work
```

> ⚠️ Keep `Σ(DB_MAX_CONNS × replicas)` across all binaries **≤ Postgres `max_connections`**.

## Layout

```
cmd/api/            entrypoint
internal/
  <feature>/        1 folder = 1 module (dto, handler, service, repository, module.go)
  shared/           config, database, cache, observability, httpx, middleware
```

## Adding a new module

See [docs/add-new-module.md](docs/add-new-module.md) — the full pattern using the `example` module as a reference.

## Worker & queue

This repo produces three binaries (`cmd/api`, `cmd/worker`, `cmd/scheduler`) from the same code. Producer/consumer uses Asynq (Redis); the scheduler uses `robfig/cron`. Details: [docs/worker-queue.md](docs/worker-queue.md).

## Built-in endpoints

- `GET /health` — health check (DB + Redis)
- `GET /metrics` — Prometheus metrics
- `/example` (the `example` module) — sample CRUD + `POST /example/produce` (enqueue); remove or replace it when starting a real service
