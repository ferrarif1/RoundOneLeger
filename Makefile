.PHONY: build run migrate test frontend windows-exe

FRONTEND_DIR ?= web
DIST_DIR ?= dist

build:
	mkdir -p bin
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

migrate:
	@echo "Run database migrations here"

test:
	go test ./...

frontend:
	npm --prefix $(FRONTEND_DIR) install --no-fund --no-audit
	npm --prefix $(FRONTEND_DIR) run build

windows-exe: frontend
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/ledger.exe ./cmd/server
