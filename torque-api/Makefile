# Torque API — developer Makefile.
#
# Opinionated targets. `make help` prints the grouped list.

SHELL          := /usr/bin/env bash
.SHELLFLAGS    := -eu -o pipefail -c
.DEFAULT_GOAL  := help

# ---- Config -----------------------------------------------------------------
DATABASE_URL    ?= postgres://torque:torque@localhost:5432/torque?sslmode=disable
MIGRATE         ?= migrate
MIGRATIONS_DIR  := ./migrations
VERSION         ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# ---- Go ---------------------------------------------------------------------
.PHONY: run
run: ## Run the API locally with dev defaults
	@if [ ! -f .env ]; then echo "  → no .env found, copying .env.example"; cp .env.example .env; fi
	@set -a; . ./.env; set +a; \
	  go run -ldflags "-X main.version=$(VERSION)" ./cmd/api

.PHONY: build
build: ## Build the binary into ./bin/torque-api
	@mkdir -p bin
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/torque-api ./cmd/api

.PHONY: test
test: ## Run unit tests (skips integration)
	go test ./... -race -count=1 -short

.PHONY: test-integration
test-integration: ## Run integration tests (require DATABASE_URL)
	DATABASE_URL="$(DATABASE_URL)" go test ./... -race -count=1 -run "Integration|Concurrent"

.PHONY: lint
lint: ## golangci-lint run (install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

# ---- Security (S52) --------------------------------------------------------
.PHONY: security
security: ## Run local security scan (vet + gosec + govulncheck). Install: go install github.com/securego/gosec/v2/cmd/gosec@latest && go install golang.org/x/vuln/cmd/govulncheck@latest
	go vet ./...
	gosec -severity=high -confidence=medium ./...
	govulncheck ./...

# ---- Migrations -------------------------------------------------------------
.PHONY: migrate-up
migrate-up: ## Apply all pending migrations
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

.PHONY: migrate-reset
migrate-reset: ## Drop everything and re-apply from zero (destroys data)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" drop -f
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

.PHONY: migrate-create
migrate-create: ## Create new migration pair (usage: make migrate-create name=add_foo)
	@if [ -z "$(name)" ]; then echo "usage: make migrate-create name=<snake_case>"; exit 1; fi
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

# ---- Seeds ------------------------------------------------------------------
.PHONY: seed-dev
seed-dev: ## Load dev seeds (marcelo@gmail.com / admin). Refuses if ENV != dev.
	@set -a; [ -f .env ] && . ./.env; set +a; \
	 if [ "$${ENV:-dev}" != "dev" ]; then \
	   echo "seed-dev: refusing to run in ENV=$${ENV}"; exit 1; \
	 fi; \
	 psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f seeds/dev_admin.sql

# ---- Docker -----------------------------------------------------------------
.PHONY: docker-up
docker-up: ## docker compose up (Postgres + migrate + API)
	docker compose up --build

.PHONY: docker-up-db
docker-up-db: ## Bring up only Postgres (run API on host with `make run`)
	docker compose up -d db

.PHONY: docker-down
docker-down: ## Stop compose stack
	docker compose down

.PHONY: docker-clean
docker-clean: ## Stop + wipe volumes (destroys dev data)
	docker compose down -v

# ---- Help -------------------------------------------------------------------
.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nTargets:\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
