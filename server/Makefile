-include .env

SERVICE=api
BINARY=bin/$(SERVICE)
MIGRATIONS_DIR=migrations
GOOSE=go tool goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)"

.PHONY: build run test lint clean seed register-demo eval-retrieval migrate-create migrate-up migrate-down migrate-status

build:
	go build -o $(BINARY) ./cmd/$(SERVICE)

run:
	go run ./cmd/$(SERVICE)

test:
	go test -race ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

seed:
	go run ./cmd/seed-local

register-demo:
	go run ./cmd/register-demo

eval-retrieval:
	go run ./cmd/eval-retrieval

migrate-create:
	$(if $(name),,$(error usage: make migrate-create name=<snake_case_name>))
	go tool goose -dir $(MIGRATIONS_DIR) create $(name) sql

migrate-up:
	$(GOOSE) up

migrate-down:
	$(GOOSE) down

migrate-status:
	$(GOOSE) status
