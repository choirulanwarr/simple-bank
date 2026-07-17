# Simple Bank — Backend Banking System

> **Production-ready backend banking system** built with Go, gRPC, PostgreSQL, and Docker. Features ACID-compliant transfers, audit trails, and clean architecture.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql)](https://www.postgresql.org/)
[![gRPC](https://img.shields.io/badge/gRPC-1.65+-4285F4?logo=grpc)](https://grpc.io/)
[![Docker](https://img.shields.io/badge/Docker-24+-2496ED?logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

---

## Features

| Feature | Description |
|---------|-------------|
| **Customer Management** | Create, read, update, deactivate customers |
| **Bank Accounts** | Multi-account per customer, auto-generated 16-digit numbers |
| **Deposits** | Atomic balance updates with transaction logging |
| **Withdrawals** | Balance validation, atomic updates, transaction records |
| **Transfers** | **ACID-compliant** inter-account transfers with deadlock prevention |
| **Audit Trail** | Automatic immutable logging via PostgreSQL triggers |
| **Authentication** | PASETO/JWT tokens with gRPC interceptors |

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.22+ |
| API | gRPC + Protocol Buffers (Buf) |
| Database | PostgreSQL 16 (pgx/v5) |
| Migrations | golang-migrate |
| SQL → Go | SQLC (compile-time safe) |
| Config | Viper (.env + env vars) |
| Auth | PASETO / JWT (golang.org/x/crypto) |
| Testing | Testify (suite + mock) |
| Container | Multi-stage Dockerfile |
| Orchestration | Docker Compose |

---

## Quick Start

### Prerequisites

```bash
go version          # 1.22+
docker --version    # 24+
docker compose version
```

### 1. Clone & Configure

```bash
git clone git@github.com:choirulanwarr/simple-bank.git
cd simple-bank/backend

# Copy env template
cp .env.example .env
# Edit .env if needed (defaults work for local Docker)
```

### 2. Start Full Stack (Database + Redis + App)

```bash
# One command: starts PostgreSQL, Redis, runs migrations, starts gRPC server
make dev

# Or step by step:
docker compose up -d        # Start PostgreSQL + Redis
make migrate-up             # Run database migrations
make server                 # Start Go server (port 9090)
```

### 3. Verify

```bash
# gRPC health check (requires grpcurl)
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check

# List services
grpcurl -plaintext localhost:9090 list
```

---

## Project Structure

```
backend/
├── api/pb/                    # Generated gRPC code
├── cmd/server/main.go         # Entry point
├── db/
│   ├── migrations/            # SQL migrations (up/down)
│   ├── queries/               # SQLC source queries
│   └── sqlc/                  # Generated Go code
├── internal/
│   ├── config/                # Viper config loader
│   ├── controller/            # Business logic layer
│   ├── middleware/            # gRPC interceptors (auth, logging, recovery)
│   ├── model/                 # Domain models & DTOs
│   ├── repository/            # Data access (wraps SQLC)
│   └── cache/                 # Redis caching (future)
├── pkg/
│   ├── password/              # Bcrypt utilities
│   └── token/                 # PASETO/JWT utilities
├── proto/simple_bank.proto    # gRPC contract
├── docker-compose.yml         # Full local stack
├── Dockerfile                 # Multi-stage build
├── Makefile                   # Common tasks
├── sqlc.yaml                  # SQLC config
└── buf.gen.yaml               # Buf config
```

---

## Key Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| **gRPC over REST** | Strongly-typed contracts, code generation, HTTP/2 performance |
| **SQLC over ORM** | Compile-time SQL validation, zero reflection, full query control |
| **PostgreSQL triggers for audit** | Immutable, cannot be bypassed by application bugs |
| **Pessimistic locking (`FOR NO KEY UPDATE`)** | Prevents lost updates in concurrent transfers |
| **Consistent lock ordering** | Mathematically prevents deadlocks (always lock lower ID first) |
| **`decimal.Decimal` for money** | Zero floating-point errors |

---

## Development Commands

```bash
# Database
make postgres          # Start PostgreSQL only
make redis             # Start Redis only
make migrate-up        # Apply migrations
make migrate-down      # Rollback last migration
make migrate-create    # Create new migration (prompts for name)

# Code Generation
make sqlc              # Generate Go from SQL
make proto             # Generate gRPC from .proto

# Testing
make test              # All unit tests with coverage
make test-integration  # Integration tests (requires Docker)

# Server
make server            # Run with hot reload (air)
make build             # Build binary to bin/server

# Docker
make docker-build      # Build image
make docker-up         # Full stack via compose
make docker-down       # Stop & remove volumes

# Quality
make fmt               # goimports formatting
make lint              # golangci-lint
```

---

## API Reference (gRPC)

### Services

```protobuf
service SimpleBank {
  // Customers
  rpc CreateCustomer(CreateCustomerRequest) returns (CreateCustomerResponse);
  rpc GetCustomer(GetCustomerRequest) returns (GetCustomerResponse);
  rpc ListCustomers(ListCustomersRequest) returns (ListCustomersResponse);
  rpc UpdateCustomer(UpdateCustomerRequest) returns (UpdateCustomerResponse);
  rpc DeleteCustomer(DeleteCustomerRequest) returns (DeleteCustomerResponse);

  // Accounts
  rpc CreateAccount(CreateAccountRequest) returns (CreateAccountResponse);
  rpc GetAccount(GetAccountRequest) returns (GetAccountResponse);
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);
  rpc UpdateAccountStatus(UpdateAccountStatusRequest) returns (UpdateAccountStatusResponse);

  // Transactions
  rpc Deposit(DepositRequest) returns (DepositResponse);
  rpc Withdraw(WithdrawRequest) returns (WithdrawResponse);
  rpc Transfer(TransferRequest) returns (TransferResponse);
  rpc GetTransactionHistory(GetTransactionHistoryRequest) returns (GetTransactionHistoryResponse);

  // Audit
  rpc GetAuditLogs(GetAuditLogsRequest) returns (GetAuditLogsResponse);

  // Auth
  rpc Login(LoginRequest) returns (LoginResponse);
}
```

### Example: Create Customer

```bash
grpcurl -plaintext -d '{"name":"John Doe","email":"john@example.com","password":"SecureP@ss1"}' \
  localhost:9090 simplebank.SimpleBank/CreateCustomer
```

### Example: Transfer

```bash
grpcurl -plaintext -d '{"from_account_id":1,"to_account_id":2,"amount":"100000.00","fee":"2500.00"}' \
  localhost:9090 simplebank.SimpleBank/Transfer
```

---

## Database Schema (ERD)

```
customers ──< accounts >── transactions
                    │
                    └──< transfers >── accounts (self-referential)
                    
All mutating tables → audit_logs (via PostgreSQL triggers)
```

Key tables:
- `customers` — id, name, email (unique), password_hash, is_active
- `accounts` — id, customer_id, account_number (unique), currency, balance (DECIMAL 18,2), status
- `transactions` — id, account_id, type (deposit/withdrawal), amount, balance_before, balance_after
- `transfers` — id, from_account_id, to_account_id, amount, fee, status (pending/completed/failed)
- `audit_logs` — table_name, record_id, operation, old_values (JSONB), new_values (JSONB), changed_by

---

## Security

- **Passwords**: bcrypt cost 12, never logged
- **Tokens**: PASETO (preferred) or JWT HS256, 15min access / 24h refresh
- **Transport**: TLS 1.3 in production (gRPC credentials)
- **SQL Injection**: 100% parameterized via SQLC
- **Secrets**: Environment variables only, `.env` in `.gitignore`

---

## Testing

```bash
# Unit tests (with real PostgreSQL test container)
make test

# Coverage report
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Integration tests (full gRPC → DB flow)
make test-integration
```

Target coverage: **≥ 80%** for `internal/` packages.

---

## Deployment

### Docker (Production)

```bash
# Build
docker build -t simple-bank:latest .

# Run (with external PostgreSQL/Redis)
docker run -d \
  -p 9090:9090 \
  -e POSTGRES_HOST=your-db-host \
  -e POSTGRES_PASSWORD=your-password \
  -e TOKEN_SYMMETRIC_KEY=your-32-byte-key \
  simple-bank:latest
```

### Kubernetes (Planned)

```
k8s/
├── namespace.yaml
├── configmap.yaml
├── secret.yaml
├── api/deployment.yaml + service.yaml + hpa.yaml + ingress.yaml
├── postgres/pvc.yaml + deployment.yaml + service.yaml
└── redis/deployment.yaml + service.yaml
```

### AWS (Planned)

- **EKS** for Kubernetes
- **RDS PostgreSQL** (Multi-AZ)
- **ElastiCache Redis**
- **ECR** for images
- **Route 53** + **ACM** + **ALB**

---

## Documentation

- **[DOC.md](DOC.md)** — Comprehensive technical documentation (Indonesian)
- **[requirements.md](requirements.md)** — Functional/non-functional requirements, user stories
- **[tasks.md](tasks.md)** — Implementation task tracker with dependencies
- **[steering/](steering/)** — Architecture decision records & standards

---

## Contributing

1. Fork the repo
2. Create feature branch: `git checkout -b feat/amazing-feature`
3. Follow code conventions: `make fmt && make lint`
4. Write tests for new functionality
5. Ensure `make test` passes
6. Submit PR with clear description

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

## Author

**Choirul Anwar** — [@choirulanwarr](https://github.com/choirulanwarr)

*Built as a learning project demonstrating modern Go backend practices.*