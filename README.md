# pamphlet-sync

`pamphlet-sync` is the Go API behind [Pamphlet](../pamphlet). It provides
Google sign-in and per-user storage for the data Pamphlet synchronizes
between devices. It also serves downloadable offline dictionary files.

## What it provides

- Google OAuth login and bearer-token sessions.
- Sync for EPUB content, reading progress, pinned words and sentences,
  notebooks, settings, per-book metadata, and the last active screen.
- Public health/version endpoints and static dictionary-file serving.

Sync routes require `Authorization: Bearer <session token>`. The API uses
PostgreSQL; handler and middleware tests use an in-memory SQLite database.

## Run locally

Requirements: Go (version in [`go.mod`](go.mod)) and Docker Compose.

```sh
cp .env.example .env
```

Set `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in `.env` to enable the
OAuth flow. The example file is configured for local development; never
use its placeholder credentials in a deployed environment.

```sh
make db-up      # start PostgreSQL
make run        # start the API on :8080
```

Check the service with `curl http://localhost:8080/healthz`. Run the
project checks with:

```sh
make check      # build, vet, formatting check, and tests
```

## Configuration and dictionaries

Configuration is read from environment variables; a local `.env` is loaded
automatically when present. See [`.env.example`](.env.example) for the
available settings. Dictionary files are served from `DICTIONARIES_DIR`
(default `./dictionaries`) and are supplied separately, not stored in this
repository. See the parent project's [dictionary and deployment guide](../DEPLOY.md).

## Project links

- Frontend: [`pamphlet`](../pamphlet)
- Routes: [`internal/routes/routes.go`](internal/routes/routes.go)
- Development conventions and sync rules: [`AGENTS.md`](AGENTS.md)
