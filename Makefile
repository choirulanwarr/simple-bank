.PHONY: postgres redis migrate migrate-up migrate-down sqlc proto test server build docker-up docker-down dev deploy-migrate deploy-install-db

# ── Database (native — PostgreSQL + Redis diinstall langsung di OS) ──
# macOS: brew install postgresql@16 redis
# Ubuntu: sudo bash deploy/install-db.sh

postgres:
	@echo "Memulai PostgreSQL native..."
	@if command -v brew &> /dev/null && brew services list 2>/dev/null | grep -q postgresql; then \
		brew services start postgresql@16 2>/dev/null || true; \
	elif pg_isready -q 2>/dev/null; then \
		echo "PostgreSQL sudah running"; \
	else \
		echo "Jalankan PostgreSQL: brew services start postgresql@16 (macOS) atau sudo systemctl start postgresql (Linux)"; \
		echo "Atau buka Postgres.app"; \
	fi

redis:
	@echo "Memulai Redis native..."
	@if command -v brew &> /dev/null; then \
		brew services start redis 2>/dev/null || true; \
	elif redis-cli ping 2>/dev/null | grep -q PONG; then \
		echo "Redis sudah running"; \
	else \
		echo "Jalankan Redis: brew services start redis (macOS) atau sudo systemctl start redis-server (Linux)"; \
	fi

# ── Migrations (Dev & Prod) ──
# Prasyarat: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate-up:
	migrate -path db/migrations -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" up

migrate-down:
	migrate -path db/migrations -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" down -all

migrate:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down-all:
	migrate -path db/migrations -database "$(DATABASE_URL)" down -all

# ── Production Deploy Targets ──

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

# ── Docker (alternative — untuk yang tetap ingin pakai container) ──

docker-up:
	docker compose up -d

docker-down:
	docker compose down -v

# ── Development (all-in-one) ──
# Pastikan PostgreSQL & Redis sudah running native sebelum menjalankan ini

dev: sqlc migrate-up server