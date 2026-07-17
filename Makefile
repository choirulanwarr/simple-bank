.PHONY: postgres redis migrate sqlc test server build docker-up docker-down dev migrate-docker

postgres:
	docker run --name postgres16 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:16-alpine

redis:
	docker run --name redis7 -p 6379:6379 -d redis:7-alpine

migrate-up:
	docker run --rm --network backend_default -v $(PWD)/db/migrations:/migrations migrate/migrate -path=/migrations -database "postgresql://root:secret@postgres:5432/simple_bank?sslmode=disable" up

migrate-down:
	docker run --rm --network backend_default -v $(PWD)/db/migrations:/migrations migrate/migrate -path=/migrations -database "postgresql://root:secret@postgres:5432/simple_bank?sslmode=disable" down -all

migrate-docker:
	@echo "Use 'make migrate-up' or 'make migrate-down' (requires docker network 'backend_default')"

sqlc:
	sqlc generate

proto:
	buf generate proto/

test:
	go test ./... -v -count=1 -cover

server:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

docker-up:
	docker compose up -d

docker-down:
	docker compose down -v

dev: postgres redis sqlc migrate-up server