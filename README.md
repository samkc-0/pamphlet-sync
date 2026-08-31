# pamphlet-sync

Go + Gin + GORM + Postgres backend for [Pamphlet](../pamphlet). Handles Google OAuth login and cross-device sync — books, reading progress, pinned words, settings, per-book metadata overrides, and navigation (which screen a user was on). Also serves the offline dictionary datasets Pamphlet's readers can download.

## Layout

```
cmd/api             entrypoint
internal/auth        Google OAuth config, session tokens
internal/config       env-based configuration + release-mode validation
internal/db           database connection + auto-migration
internal/models       GORM models
internal/handlers     gin handlers
internal/middleware   session auth middleware
internal/routes       route registration
internal/version      build-time commit/timestamp metadata
```

## Routes

Public:

| Route | |
|---|---|
| `GET /healthz` | liveness + DB check |
| `GET /version` | deployed commit SHA + build time |
| `GET /dictionaries/*` | static offline dictionary files (see below) |
| `GET /auth/google/login`, `GET /auth/google/callback` | OAuth flow, hands a session token back to the frontend via `postMessage` |

Bearer-authenticated (`Authorization: Bearer <session token>`):

| Route | |
|---|---|
| `GET /me`, `POST /auth/logout` | current user / end session |
| `POST /books`, `GET /books`, `GET /books/:hash`, `POST /books/:hash/delete` | book content, keyed by content hash |
| `POST /progress/:hash`, `GET /progress` | reading progress |
| `POST /pinned-words`, `GET /pinned-words` | pinned vocabulary |
| `POST /settings`, `GET /settings` | user settings |
| `POST /book-metadata/:hash`, `GET /book-metadata` | per-book metadata overrides |
| `POST /navigation`, `GET /navigation` | last-active screen |

## Getting started

```bash
cp .env.example .env
make db-up      # starts Postgres via docker-compose
make run        # starts the API on :8080
make check      # build, vet, gofmt, test — run before every commit
```

Check it's alive:

```bash
curl localhost:8080/healthz
```

## Offline dictionaries

`GET /dictionaries/*` serves trimmed Wiktionary datasets read from `DICTIONARIES_DIR` (default `./dictionaries`, bind-mounted to `/app/dictionaries` in the sandbox deploy). The datasets themselves are built and uploaded from outside this repo — see `pamphlet-project/scripts/dictionary/` and `pamphlet-project/DEPLOY.md`.

## Adding a model

1. Define the struct in `internal/models`.
2. Add it to the slice returned by `models.All()` so it's picked up by auto-migration.
3. Add handlers under `internal/handlers` and register routes in `internal/routes/routes.go`.
4. Add tests — see `AGENTS.md` for the sync-feature convention (every new synced field needs both a backend endpoint and matching frontend push/pull logic in `pamphlet`).

## Related

- Frontend: sibling repo `pamphlet`.
- Conventions, branch workflow, and known debt: `AGENTS.md`.
- Sandbox deploy (Docker Compose, shared Caddy, GitHub Actions): `pamphlet-project/DEPLOY.md`.
