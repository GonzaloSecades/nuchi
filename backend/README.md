# Nuchi Go Backend

This directory contains the separate Go API that will replace the current
Hono/Drizzle/Neon backend.

The scaffold currently exposes only a health endpoint. Database access, auth,
OpenAPI generation, migrations, and frontend integration are intentionally left
to their own issues.

## Local Run

```bash
cd backend
go run ./cmd/api
```

The API listens on `0.0.0.0:8080` by default.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `BACKEND_HOST` | `0.0.0.0` | HTTP listen host |
| `BACKEND_PORT` | `8080` | HTTP listen port |

## Health Check

```bash
curl http://localhost:8080/api/health
```

Expected response:

```json
{
  "service": "nuchi-api",
  "status": "ok",
  "time": "2026-06-29T00:00:00Z"
}
```

## Verification

```bash
cd backend
go test ./...
go run ./cmd/api
```
