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

For a feature that needs real isolation from whatever else is happening on
`dev`, use a **git worktree**:

```
git worktree add ../feature-name -b feature-name
```

Checks `feature-name` out into a sibling directory
(`pamphlet-project/feature-name`), with its own working tree. Go's module
cache is shared machine-wide, so unlike the frontend there's no per-worktree
install step — `go build`/`go test` just work. `deploy/.env` isn't
committed, so a worktree used to deploy directly needs its own copy.

Merge into `dev` from either worktree, then clean up:

```
git worktree remove ../feature-name
git branch -d feature-name
```

`git worktree list` flags a worktree as `prunable` if its directory got
deleted without `remove` first — clear those with `git worktree prune`.

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

## Cross-device sync

Signed-in users sync book content, reading progress, and pinned words
across devices (`internal/handlers/books.go`, `progress.go`,
`pinned_words.go`). **Whenever you add or change frontend state that should
follow the user across devices, it needs a matching piece here** — a model
(scoped per-user; composite-keyed on whatever the state is naturally keyed
by, e.g. a book's content hash) and a protected POST/GET endpoint pair
following the existing shape:

- Writes are a *conditional* last-write-wins: `First` the existing row,
  compare its `UpdatedAt` against the incoming client-supplied timestamp,
  no-op if the incoming write isn't newer. Never a blind upsert (GORM's
  `Save` silently updates zero rows if the composite key doesn't exist yet —
  branch on `gorm.ErrRecordNotFound` explicitly instead).
- A "deleted" or "unpinned" state is a row update (a boolean/flag field),
  never a row delete — deleting the row destroys the timestamp the next
  conflicting write needs to compare against.
- `List` endpoints are scoped to the current user and never return a book's
  `Content` (see `BookHandler.List`'s explicit `.Select()`).

Conversely, don't add a sync endpoint here without also wiring the
frontend's push/pull for it (see the matching note in `pamphlet`'s
AGENTS.md) — a backend-only or frontend-only half of a sync feature is
worse than not having it, since it silently doesn't do anything.

## Known debt

- `AutoMigrate` runs on every boot instead of a real migration tool. Fine
  for the current simple, single-operator schema; it can't safely
  drop/rename columns, so plan to replace it before schema changes get
  less trivial.
