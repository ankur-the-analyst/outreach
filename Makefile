DATABASE_URL ?= postgres://outreach:outreach@127.0.0.1:5433/outreach?sslmode=disable
DATABASE_APP_URL ?= postgres://outreach_app:outreach_app@127.0.0.1:5433/outreach?sslmode=disable
MAILPIT_SMTP ?= 127.0.0.1:1025
MAILPIT_API ?= http://127.0.0.1:8025
MASTER_KEYS ?= 1:0000000000000000000000000000000000000000000000000000000000000001
MASTER_KEY_ACTIVE ?= 1
DBMATE := dbmate --url "$(DATABASE_URL)" --migrations-dir backend/migrations

.PHONY: up down logs api worker migrate migrate-down migrate-status sqlc test test-int web

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f postgres

api:
	cd backend && DATABASE_APP_URL="$(DATABASE_APP_URL)" MASTER_KEYS="$(MASTER_KEYS)" MASTER_KEY_ACTIVE="$(MASTER_KEY_ACTIVE)" ADDR="$(or $(ADDR),:8080)" go run ./cmd/api

worker:
	cd backend && DATABASE_APP_URL="$(DATABASE_APP_URL)" MASTER_KEYS="$(MASTER_KEYS)" MASTER_KEY_ACTIVE="$(MASTER_KEY_ACTIVE)" go run ./cmd/worker

sqlc:
	cd backend && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate

test:
	cd backend && go test ./...

test-int:
	cd backend && DATABASE_APP_URL="$(DATABASE_APP_URL)" MASTER_KEYS="$(MASTER_KEYS)" MAILPIT_SMTP="$(MAILPIT_SMTP)" MAILPIT_API="$(MAILPIT_API)" go test ./... -count=1 -timeout 60s

web:
	cd apps/web && pnpm dev

migrate:
	$(DBMATE) up

migrate-down:
	$(DBMATE) down

migrate-status:
	$(DBMATE) status
