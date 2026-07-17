-- Customers
CREATE TABLE "customers" (
  "id"            BIGSERIAL PRIMARY KEY,
  "name"          VARCHAR(255) NOT NULL,
  "email"         VARCHAR(255) UNIQUE NOT NULL,
  "password_hash" VARCHAR(255) NOT NULL,
  "is_active"     BOOLEAN NOT NULL DEFAULT TRUE,
  "created_at"    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  "updated_at"    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Accounts
CREATE TABLE "accounts" (
  "id"              BIGSERIAL PRIMARY KEY,
  "customer_id"     BIGINT NOT NULL REFERENCES "customers" ("id"),
  "account_number"  VARCHAR(20) UNIQUE NOT NULL,
  "currency"        VARCHAR(3) NOT NULL DEFAULT 'IDR',
  "balance"         DECIMAL(18, 2) NOT NULL DEFAULT 0.00,
  "status"          VARCHAR(20) NOT NULL DEFAULT 'active'
                    CHECK ("status" IN ('active', 'inactive', 'closed')),
  "created_at"      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  "updated_at"      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Transactions
CREATE TABLE "transactions" (
  "id"              BIGSERIAL PRIMARY KEY,
  "account_id"      BIGINT NOT NULL REFERENCES "accounts" ("id"),
  "type"            VARCHAR(20) NOT NULL
                    CHECK ("type" IN ('deposit', 'withdrawal')),
  "amount"          DECIMAL(18, 2) NOT NULL CHECK ("amount" > 0),
  "balance_before"  DECIMAL(18, 2) NOT NULL,
  "balance_after"   DECIMAL(18, 2) NOT NULL,
  "reference"       VARCHAR(100),
  "description"     TEXT,
  "created_at"      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Transfers
CREATE TABLE "transfers" (
  "id"                BIGSERIAL PRIMARY KEY,
  "from_account_id"   BIGINT NOT NULL REFERENCES "accounts" ("id"),
  "to_account_id"     BIGINT NOT NULL REFERENCES "accounts" ("id"),
  "amount"            DECIMAL(18, 2) NOT NULL CHECK ("amount" > 0),
  "fee"               DECIMAL(18, 2) NOT NULL DEFAULT 0.00,
  "status"            VARCHAR(20) NOT NULL DEFAULT 'pending'
                      CHECK ("status" IN ('pending', 'completed', 'failed')),
  "reference"         VARCHAR(100),
  "description"       TEXT,
  "created_at"        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  "completed_at"      TIMESTAMPTZ
);

-- Audit Logs
CREATE TABLE "audit_logs" (
  "id"          BIGSERIAL PRIMARY KEY,
  "table_name"  VARCHAR(100) NOT NULL,
  "record_id"   BIGINT NOT NULL,
  "operation"   VARCHAR(10) NOT NULL
                CHECK ("operation" IN ('INSERT', 'UPDATE', 'DELETE')),
  "old_values"  JSONB,
  "new_values"  JSONB,
  "changed_by"  VARCHAR(255),
  "changed_at"  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_customers_email ON "customers" ("email");
CREATE INDEX idx_accounts_customer_id ON "accounts" ("customer_id");
CREATE INDEX idx_accounts_account_number ON "accounts" ("account_number");
CREATE INDEX idx_transactions_account_id ON "transactions" ("account_id");
CREATE INDEX idx_transactions_created_at ON "transactions" ("created_at");
CREATE INDEX idx_transfers_from_account ON "transfers" ("from_account_id");
CREATE INDEX idx_transfers_to_account ON "transfers" ("to_account_id");
CREATE INDEX idx_transfers_status ON "transfers" ("status");
CREATE INDEX idx_audit_logs_table_record ON "audit_logs" ("table_name", "record_id");
CREATE INDEX idx_audit_logs_changed_at ON "audit_logs" ("changed_at");