.PHONY: run build test check db-up db-down

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

check:
	go build ./...
	go vet ./...
	test -z "$$(gofmt -l .)"
	go test ./...

db-up:
	docker compose up -d

db-down:
	docker compose down
