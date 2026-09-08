.PHONY: all
all: fmt test build-server build-cleanup build-client

BINARY_SERVER=gophkeeper-server
BINARY_CLEANUP=gophkeeper-cleanup
BINARY_CLIENT=gophkeeper-client
VERSION ?= 0.0.1
BUILD_DATE = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)"
API_SPEC = $(shell pwd)/api/openapi.yml
GENERATED_DIR = $(shell pwd)/pkg/api/generated

.PHONY: fmt
fmt:
	gofmt -w ./cmd ./internal

.PHONY: test
test:
	go test -race -coverprofile=cover.profile -v ./internal/...
	go tool cover -func=cover.profile
	rm -f cover.profile

.PHONY: build-server
build-server:
	go build -race $(LDFLAGS) -o bin/$(BINARY_SERVER) ./cmd/$(BINARY_SERVER)

.PHONY: build-cleanup
build-cleanup:
	go build -race -o bin/$(BINARY_CLEANUP) ./cmd/$(BINARY_CLEANUP)

.PHONY: build-client
build-client:
	go build -race $(LDFLAGS) -o bin/$(BINARY_CLIENT) ./cmd/$(BINARY_CLIENT)

.PHONY: run-server-dev
run-server-dev:
	go run -race ./cmd/$(BINARY_SERVER)

.PHONY: run-client-dev
run-client-dev:
	go run -race ./cmd/$(BINARY_CLIENT)

.PHONY: cleanup-blobs
cleanup-blobs:
	go run -race ./cmd/$(BINARY_CLEANUP)

.PHONY: compose-up
compose-up:
	docker compose up -d postgres

.PHONY: compose-down
compose-down:
	docker compose down

.PHONY: migrate-up
migrate-up:
	docker compose exec -T postgres psql -U gophkeeper -d gophkeeper -f /dev/stdin < migrations/000001_create_users.up.sql
	docker compose exec -T postgres psql -U gophkeeper -d gophkeeper -f /dev/stdin < migrations/000002_create_secrets.up.sql

.PHONY: migrate-down
migrate-down:
	docker compose exec -T postgres psql -U gophkeeper -d gophkeeper -f /dev/stdin < migrations/000002_create_secrets.down.sql
	docker compose exec -T postgres psql -U gophkeeper -d gophkeeper -f /dev/stdin < migrations/000001_create_users.down.sql

.PHONY: generate-api
generate-api:
	mkdir -p $(GENERATED_DIR)/infra $(GENERATED_DIR)/auth $(GENERATED_DIR)/secrets
	oapi-codegen -config api/configs/infra.yml $(API_SPEC) > $(GENERATED_DIR)/infra/infra.gen.go
	oapi-codegen -config api/configs/auth.yml $(API_SPEC) > $(GENERATED_DIR)/auth/auth.gen.go
	oapi-codegen -config api/configs/secrets.yml $(API_SPEC) > $(GENERATED_DIR)/secrets/secrets.gen.go
