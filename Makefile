.PHONY: postgres redis migrate migrate-up migrate-down sqlc proto test server build docker-up docker-down dev deploy-migrate deploy-install-db

# ── Local Development (Docker-based PG + Redis) ──

postgres:
	docker run --name postgres16 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:16-alpine

redis:
	docker run --name redis7 -p 6379:6379 -d redis:7-alpine

migrate-up:
	migrate -path db/migrations -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" up

migrate-down:
	migrate -path db/migrations -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" down -all

# ── Native PostgreSQL (untuk dev yang pakai native atau production) ──
# Jalankan setelah PostgreSQL diinstall native di VPS
# Pastikan binary migrate terinstall: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down-all:
	migrate -path db/migrations -database "$(DATABASE_URL)" down -all

# ── Production Deploy Targets ──
# Digunakan di VPS setelah PG & Redis native terinstall

deploy-install-db:
	sudo bash deploy/install-db.sh

deploy-migrate:
	DATABASE_URL="postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" \
	migrate -path db/migrations -database "$$DATABASE_URL" up

deploy-rollback:
	DATABASE_URL="postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" \
	migrate -path db/migrations -database "$$DATABASE_URL" down 1

# ── Code Generation ──

sqlc:
	sqlc generate

proto:
	buf generate proto/

# ── Testing ──

test:
	go test ./... -v -count=1 -cover

# ── Build & Run ──

server:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

# ── Docker (development stack — includes PG + Redis) ──

docker-up:
	docker compose up -d

docker-down:
	docker compose down -v

# ── Development (all-in-one) ──

dev: postgres redis sqlc migrate-up server