.PHONY: postgres redis migrate sqlc test server build docker-up docker-down dev

postgres:
	docker run --name postgres16 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:16-alpine

redis:
	docker run --name redis7 -p 6379:6379 -d redis:7-alpine

migrate-up:
	migrate -path db/migrations -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" up

migrate-down:
	migrate -path db/migrations -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" down

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