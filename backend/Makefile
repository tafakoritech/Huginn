SHELL := /bin/bash
APP ?= huginn
MIGRATIONS_DIR ?= migrations
ENV_FILE ?= .env

.PHONY: help dev build test lint migrate-up migrate-down migrate-status seed backfill-contacts logs compose-up compose-down clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

compose-up: ## Start all infrastructure (postgres, redis, rabbitmq, minio, tika)
	docker compose up -d

compose-down: ## Stop infrastructure
	docker compose down

dev: ## Run the API locally (hot paths)
	go run ./cmd/api

worker: ## Run the worker locally
	go run ./cmd/worker

build: ## Build all binaries
	mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o bin/huginn-api ./cmd/api
	go build -trimpath -ldflags="-s -w" -o bin/huginn-worker ./cmd/worker
	go build -trimpath -ldflags="-s -w" -o bin/huginn-cli ./cmd/cli

test: ## Run all tests
	go test ./...

lint: ## Vet and staticcheck
	go vet ./...
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed; running go vet only"

migrate-up: ## Apply database migrations
	@test -f $(ENV_FILE) && set -a && source $(ENV_FILE) && set +a; \
	MIGRATIONS_DIR=$(MIGRATIONS_DIR) go run ./cmd/cli up

migrate-down: ## Roll back the last migration
	@test -f $(ENV_FILE) && set -a && source $(ENV_FILE) && set +a; \
	MIGRATIONS_DIR=$(MIGRATIONS_DIR) go run ./cmd/cli down

migrate-status: ## Show migration status
	@test -f $(ENV_FILE) && set -a && source $(ENV_FILE) && set +a; \
	MIGRATIONS_DIR=$(MIGRATIONS_DIR) go run ./cmd/cli status

seed: ## Seed development data
	@test -f $(ENV_FILE) && set -a && source $(ENV_FILE) && set +a; \
	MIGRATIONS_DIR=$(MIGRATIONS_DIR) go run ./cmd/cli seed

backfill-contacts: ## Safely recover explicit phone/email from source docs for leads missing contact data
	@test -f $(ENV_FILE) && set -a && source $(ENV_FILE) && set +a; \
	MIGRATIONS_DIR=$(MIGRATIONS_DIR) go run ./cmd/cli backfill-contacts

logs: ## Tail api + worker logs
	docker compose logs -f huginn-api huginn-worker

clean: ## Remove build artifacts
	rm -rf bin
