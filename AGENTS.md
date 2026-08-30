# pamphlet-sync (backend)

Go + Gin + GORM + Postgres backend for Pamphlet. Currently just Google
OAuth login and opaque bearer-token sessions; the frontend is the sibling
repo `pamphlet`.

## Branch workflow

Work happens on `dev`. **Never push or merge to `main`** — that is done
manually by the maintainer.

**Pushing to `dev` is not a no-op**: `.github/workflows/deploy-sandbox.yml`
runs the test suite and, if it passes, redeploys the live sandbox server
over SSH. Don't push half-finished work to `dev`.

## Before committing

```
make check   # go build ./... && go vet ./... && gofmt -l . && go test ./...
```

Run this before every commit — it's the same set of checks CI runs before
allowing a deploy.

## Tests

`internal/config`, `internal/auth`, `internal/middleware`, and
`internal/handlers` have real test coverage, focused on the
security-critical paths: session validation/expiry in
`middleware.RequireSession`, the OAuth CSRF-state check in
`GoogleCallback`, and the `Config.Validate()` fail-fast logic. Middleware/
handler tests use an in-memory SQLite DB (`github.com/glebarez/sqlite`, pure
Go, no CGO) rather than a live Postgres, so they run fast with no external
services. Extend this coverage as new handlers get added — don't let it
regress back to zero.

## Config

All configuration is env vars, loaded in `internal/config/config.go` (a
local `.env` is loaded automatically if present; not committed).
`Config.Validate()` refuses to boot in `GIN_MODE=release` if OAuth secrets
are missing or key URLs (`FRONTEND_URL`, `DATABASE_URL`,
`GOOGLE_REDIRECT_URL`) still point at `localhost` — a prod-shaped `.env`
with dev-default values will now fail fast at startup by design.

## Deploy

Sandbox deploy is a shared-Caddy setup on an external `appnet` Docker
network (see `../DEPLOY.md`). The `api` service's `container_name:
pamphlet-sync` in `deploy/docker-compose.yml` must stay exactly that — a
separate Caddyfile, in another repo, hardcodes that hostname for routing.

## Known debt

- `AutoMigrate` runs on every boot instead of a real migration tool. Fine
  for the current simple, single-operator schema; it can't safely
  drop/rename columns, so plan to replace it before schema changes get
  less trivial.
