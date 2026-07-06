APP_NAME       ?= arche
BIN_DIR        ?= bin
MIGRATIONS_DIR ?= migrations
POSTGRES_DSN   ?= postgres://postgres:postgres@localhost:5432/app?sslmode=disable

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "} {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: generate
generate: ## Generate server code from the OpenAPI spec
	go tool oapi-codegen -config generated/api/definitions-cfg.yaml generated/api/definitions.yaml
	go tool oapi-codegen -config generated/api/cfg.yaml generated/api/api.yaml

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

.PHONY: build
build: ## Build the binary into $(BIN_DIR)/app
	mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(BIN_DIR)/app ./cmd/app

.PHONY: build-cli
build-cli: ## Build the CLI binary into $(BIN_DIR)/cli
	mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(BIN_DIR)/cli ./cmd/cli

.PHONY: run
run: ## Run the app locally (loads .env from the project root)
	go run ./cmd/app

.PHONY: test
test: ## Run tests with race detector
	go test -race -count=1 ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format and vet
	gofmt -s -w .
	go vet ./...

.PHONY: up
up: ## Start the docker-compose stack
	docker compose up -d --build

.PHONY: down
down: ## Stop the docker-compose stack
	docker compose down

.PHONY: migrate-up
migrate-up: ## Apply all migrations up
	migrate -path $(MIGRATIONS_DIR) -database "$(POSTGRES_DSN)" up

.PHONY: migrate-down
migrate-down: ## Roll migrations one step down
	migrate -path $(MIGRATIONS_DIR) -database "$(POSTGRES_DSN)" down 1

.PHONY: migrate-new
migrate-new: ## Create a new migration; usage: make migrate-new name=add_xxx
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

.PHONY: clean
clean: ## Remove the build directory
	rm -rf $(BIN_DIR)
