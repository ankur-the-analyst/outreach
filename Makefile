DATABASE_URL ?= postgres://outreach:outreach@127.0.0.1:5433/outreach?sslmode=disable
DBMATE := dbmate --url "$(DATABASE_URL)" --migrations-dir backend/migrations

.PHONY: up down logs api migrate migrate-down migrate-status

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f postgres

api:
	cd backend && go run ./cmd/api

migrate:
	$(DBMATE) up

migrate-down:
	$(DBMATE) down

migrate-status:
	$(DBMATE) status
