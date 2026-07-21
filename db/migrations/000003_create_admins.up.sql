CREATE TABLE "admins" (
  "id"            BIGSERIAL PRIMARY KEY,
  "name"          VARCHAR(255) NOT NULL,
  "email"         VARCHAR(255) UNIQUE NOT NULL,
  "password_hash" VARCHAR(255) NOT NULL,
  "role"          VARCHAR(50) NOT NULL DEFAULT 'admin'
                  CHECK ("role" IN ('superadmin', 'admin', 'viewer')),
  "is_active"     BOOLEAN NOT NULL DEFAULT TRUE,
  "created_at"    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  "updated_at"    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admins_email ON "admins" ("email");
