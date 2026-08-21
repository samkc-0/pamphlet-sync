.PHONY: run build db-up db-down

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

db-up:
	docker compose up -d

db-down:
	docker compose down
