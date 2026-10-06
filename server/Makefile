SERVICE=api
BINARY=bin/$(SERVICE)

.PHONY: build run test lint clean seed

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
