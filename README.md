# Simple Bank — Backend Banking System

> **Backend perbankan sederhana** dengan Go, gRPC, PostgreSQL, Docker. Mendemonstrasikan implementasi transaksi finansial ACID-compliant, audit trail, dan arsitektur microservice-ready.

---

## ✨ Fitur Utama

| Fitur | Deskripsi |
|-------|-----------|
| **Customer Management** | CRUD nasabah, validasi email unik, password hashing (bcrypt) |
| **Account Management** | Multi-rekening per nasabah, auto-generate nomor rekening 16 digit |
| **Deposit / Withdrawal** | Update saldo atomik, pencatatan transaksi dengan balance_before/after |
| **Transfer Dana** | ACID transaction, pessimistic locking (`FOR NO KEY UPDATE`), deadlock-safe |
| **Audit Trail** | Trigger PostgreSQL otomatis (INSERT/UPDATE/DELETE) — immutable |
| **Authentication** | PASETO/JWT token, gRPC auth interceptor |

---

## 🛠 Tech Stack

| Layer | Technology |
|-------|------------|
| **Language** | Go 1.22+ |
| **API** | gRPC + Protocol Buffers (Buf) |
| **Database** | PostgreSQL 16 (pgx/v5 driver) |
| **SQL → Go** | SQLC (compile-time type-safe) |
| **Migrations** | golang-migrate |
| **Config** | Viper (.env + env vars) |
| **Testing** | Testify (suite + mock) |
| **Container** | Docker multi-stage, Docker Compose |
| **Cache** | Redis 7 (planned) |

---

## 🚀 Quick Start

### Prasyarat

```bash
go version          # Go 1.22+
docker --version    # Docker 24+
docker compose version
```

### 1. Clone & Setup

```bash
git clone git@github.com:choirulanwar/simple-bank.git
cd simple-bank/backend

# Copy env template
cp .env.example .env
# Edit .env jika perlu (default sudah OK untuk local)
```

### 2. Jalankan Semua Service (1 command)

```bash
# Start PostgreSQL, Redis, run migrations, start API server
make dev
```

Atau manual via Docker Compose:

```bash
docker compose up -d
# API: localhost:9090 (gRPC)
# PostgreSQL: localhost:5432
# Redis: localhost:6379
```

### 3. Verifikasi

```bash
# Health check via grpcurl
grpcurl -plaintext localhost:9090 list
# simplebank.SimpleBank
```

---

## 📁 Struktur Project

```
backend/
├── api/pb/                 # Generated gRPC code
├── cmd/server/main.go      # Entry point
├── db/
│   ├── migrations/         # SQL migrations (up/down)
│   ├── queries/            # SQLC source queries
│   └── sqlc/               # Generated Go code
├── internal/
│   ├── config/             # Viper config loader
│   ├── controller/         # Business logic
│   ├── middleware/         # gRPC interceptors
│   ├── model/              # Domain models
│   └── repository/         # Data access (wraps SQLC)
├── pkg/
│   ├── password/           # bcrypt utilities
│   └── token/              # PASETO/JWT utilities
├── proto/simple_bank.proto # gRPC contract
├── docker-compose.yml      # Full stack (PG + Redis + App + Migrate)
├── Dockerfile              # Multi-stage production build
├── Makefile                # Common tasks
├── sqlc.yaml               # SQLC config
└── buf.gen.yaml            # Buf config
```

---

## 🔧 Development Commands

```bash
# Database
make postgres          # Start PostgreSQL only
make redis             # Start Redis only
make migrate-up        # Run migrations
make migrate-down      # Rollback last migration
make migrate-create    # Create new migration (interactive)

# Code Generation
make sqlc              # Generate Go code from SQL
make proto             # Generate gRPC code from .proto
make generate          # Both sqlc + proto

# Testing
make test              # All unit tests with coverage
make test-integration  # Integration tests (requires Docker)

# Build & Run
make server            # Run server locally (hot reload via air)
make build             # Build binary to bin/server
make docker-build      # Build Docker image

# Docker
make docker-up         # docker compose up -d
make docker-down       # docker compose down -v

# Code Quality
make lint              # golangci-lint
make fmt               # goimports -w .
```

---

## 🗄 Database Schema (ERD)

```
customers ◄──── accounts ► transactions
  │              │
  │              └──── transfers ◄──── accounts
  │
  └──── audit_logs (trigger-based, all tables)
```

**5 Tabel Utama:**
- `customers` — nasabah (id, name, email, password_hash, is_active)
- `accounts` — rekening (customer_id, account_number, currency, balance, status)
- `transactions` — deposit/withdrawal per rekening
- `transfers` — transfer antar rekening (from_account, to_account, amount, fee, status)
- `audit_logs` — immutable audit trail (JSONB old/new values)

---

## 🔐 Security Highlights

- **Password**: bcrypt cost 12, never logged
- **Token**: PASETO (preferred) / JWT HS256, 15min access / 24h refresh
- **Money**: `decimal.Decimal` — zero floating-point errors
- **SQL Injection**: 100% SQLC parameterized queries
- **Audit**: Database-level triggers — cannot be bypassed by app code

---

## 📡 gRPC API (Protobuf)

Service: `simplebank.SimpleBank`

| RPC | Request | Response |
|-----|---------|----------|
| `CreateCustomer` | name, email, password | Customer |
| `GetCustomer` | id | Customer |
| `ListCustomers` | limit, offset | Customer[] + pagination |
| `CreateAccount` | customer_id, currency | Account |
| `GetAccount` | id | Account |
| `ListAccounts` | customer_id | Account[] |
| `Deposit` | account_id, amount, ref, desc | Transaction + balance_after |
| `Withdraw` | account_id, amount, ref, desc | Transaction + balance_after |
| `Transfer` | from_account_id, to_account_id, amount, fee | TransferResponse (from/to account state) |
| `GetTransactionHistory` | account_id, limit, offset | Transaction[] + pagination |
| `GetAuditLogs` | table_name, record_id | AuditLog[] |
| `Login` | email, password | access_token, expires_at |

---

## 🧪 Testing

```bash
# Unit tests (real PostgreSQL test container)
go test ./internal/repository/... -v -count=1

# All tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Integration tests (full gRPC → DB)
go test ./test/integration/... -v -tags=integration
```

**Coverage target:** ≥ 80% untuk `internal/repository/` dan `internal/controller/`

---

## 🐳 Production Deployment

### Docker Image

```bash
docker build -t simple-bank:latest .
docker run -p 9090:9090 --env-file .env simple-bank:latest
```

- Multi-stage build (builder → alpine runtime)
- Non-root user (`appuser`)
- Image size < 25MB
- Healthcheck endpoint included

### Kubernetes (Planned)

```
k8s/
├── namespace.yaml
├── configmap.yaml
├── secret.yaml
├── api/deployment.yaml + service.yaml + hpa.yaml + ingress.yaml
├── postgres/ (PVC, Deployment, Service)
└── redis/  (Deployment, Service)
```

### AWS (Planned)

- **EKS** — Managed Kubernetes
- **RDS PostgreSQL** — Multi-AZ, automated backup
- **ElastiCache Redis** — Managed Redis
- **ECR** — Container registry
- **ALB** — Load balancer
- **Secrets Manager** — DB passwords, token keys

---

## 📚 Dokumentasi Lengkap

- **[DOC.md](DOC.md)** — Panduan teknis lengkap (database, SQLC, gRPC, Docker, K8s, AWS)
- **[requirements.md](requirements.md)** — Functional & non-functional requirements, user stories
- **[tasks.md](tasks.md)** — Implementation task tracking dengan dependency graph
- **[steering/](steering/)** — Architecture decisions, code conventions, API standards, security policies

---

## 🤝 Contributing

1. Fork repo
2. Buat feature branch: `git checkout -b feature/nama-fitur`
3. Commit changes: `git commit -m "feat: deskripsi singkat"`
4. Push: `git push origin feature/nama-fitur`
5. Buat Pull Request

**Code style:** `goimports`, `golangci-lint` — run `make fmt && make lint` sebelum commit.

---

## 📄 License

MIT License — bebas digunakan, dimodifikasi, didistribusikan.

---

## 👨‍💻 Author

**Choirul Anwar** — [@choirulanwar](https://github.com/choirulanwar)

*Project ini dibuat sebagai learning project & portfolio backend banking system.*