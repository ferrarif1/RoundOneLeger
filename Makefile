.PHONY: build run migrate test

build:
	mkdir -p bin
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

migrate:
	@echo "Run database migrations here"

test:
	go test ./...
