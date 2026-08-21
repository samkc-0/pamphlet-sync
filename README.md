# pamphlet-sync

Go + Gin + GORM + Postgres API boilerplate.

## Layout

```
cmd/api            entrypoint
internal/config     env-based configuration
internal/db         database connection + auto-migration
internal/models     GORM models
internal/handlers   gin handlers
internal/routes     route registration
```

## Getting started

```bash
cp .env.example .env
make db-up      # starts Postgres via docker-compose
make run        # starts the API on :8080
```

Check it's alive:

```bash
curl localhost:8080/healthz
```

## Adding a model

1. Define the struct in `internal/models`.
2. Add it to the slice returned by `models.All()` so it's picked up by auto-migration.
3. Add handlers under `internal/handlers` and register routes in `internal/routes/routes.go`.
