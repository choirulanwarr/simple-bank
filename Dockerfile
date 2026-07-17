# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Install tools
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
RUN go install github.com/bufbuild/buf/cmd/buf@latest

# Generate code
RUN sqlc generate
RUN buf generate proto/

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Stage 2: Run
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /server .
COPY --from=builder /app/.env .env

USER appuser

EXPOSE 9090

ENTRYPOINT ["./server"]