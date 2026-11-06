DATABASE_URL ?= postgres://outreach:outreach@127.0.0.1:5433/outreach?sslmode=disable
DATABASE_APP_URL ?= postgres://outreach_app:outreach_app@127.0.0.1:5433/outreach?sslmode=disable
MAILPIT_SMTP ?= 127.0.0.1:1025
MAILPIT_API ?= http://127.0.0.1:8025
DBMATE := dbmate --url "$(DATABASE_URL)" --migrations-dir backend/migrations

.PHONY: up down logs api migrate migrate-down migrate-status sqlc test test-int web

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f postgres

api:
	cd backend && DATABASE_APP_URL="$(DATABASE_APP_URL)" ADDR="$(or $(ADDR),:8080)" go run ./cmd/api

sqlc:
	cd backend && go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate

test:
	cd backend && go test ./...

test-int:
	cd backend && DATABASE_APP_URL="$(DATABASE_APP_URL)" MAILPIT_SMTP="$(MAILPIT_SMTP)" MAILPIT_API="$(MAILPIT_API)" go test ./... -count=1 -timeout 60s

web:
	cd apps/web && pnpm dev

migrate:
	$(DBMATE) up

migrate-down:
	$(DBMATE) down

migrate-status:
	$(DBMATE) status
