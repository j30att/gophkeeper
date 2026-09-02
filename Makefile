.PHONY: all
all: fmt test build-server

BINARY_SERVER=gophkeeper-server
VERSION ?= 0.0.1
BUILD_DATE = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

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

.PHONY: run-server-dev
run-server-dev:
	go run -race ./cmd/$(BINARY_SERVER)

.PHONY: compose-up
compose-up:
	docker compose up -d postgres

.PHONY: compose-down
compose-down:
	docker compose down

.PHONY: migrate-up
migrate-up:
	docker compose exec -T postgres psql -U gophkeeper -d gophkeeper -f /dev/stdin < migrations/000001_create_users.up.sql

.PHONY: migrate-down
migrate-down:
	docker compose exec -T postgres psql -U gophkeeper -d gophkeeper -f /dev/stdin < migrations/000001_create_users.down.sql

.PHONY: generate-api
generate-api:
	mkdir -p pkg/api/generated/auth
	oapi-codegen -config api/configs/auth.yml api/openapi.yml > pkg/api/generated/auth/auth.gen.go
