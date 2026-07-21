#!/bin/bash
# install-db.sh — Install PostgreSQL 16 + Redis 7 natively on Ubuntu/Debian VPS
# Usage: sudo bash deploy/install-db.sh
set -euo pipefail

echo "=== Installing PostgreSQL 16 ==="
# Add PostgreSQL official repo
sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg

sudo apt-get update -qq
sudo apt-get install -y -qq postgresql-16 postgresql-client-16

# Start & enable on boot
sudo systemctl enable postgresql
sudo systemctl start postgresql

# Create database user + database (ganti password sesuai .env)
DB_USER="${POSTGRES_USER:-root}"
DB_PASS="${POSTGRES_PASSWORD:-secret}"
DB_NAME="${POSTGRES_DB:-simple_bank}"

sudo -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASS' SUPERUSER;" 2>/dev/null || echo "User $DB_USER already exists"
sudo -u postgres psql -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>/dev/null || echo "Database $DB_NAME already exists"

echo "=== PostgreSQL 16 installed ==="

echo "=== Installing Redis 7 ==="
curl -fsSL https://packages.redis.io/gpg | sudo gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/redis.list

sudo apt-get update -qq
sudo apt-get install -y -qq redis

# Start & enable on boot
sudo systemctl enable redis-server
sudo systemctl start redis-server

echo "=== Redis 7 installed ==="

# Tuning PostgreSQL untuk 1GB RAM
echo "=== Tuning PostgreSQL ==="
PG_CONF="/etc/postgresql/16/main/postgresql.conf"
sudo sed -i "s/^shared_buffers = .*/shared_buffers = 256MB/" "$PG_CONF"
sudo sed -i "s/^effective_cache_size = .*/effective_cache_size = 512MB/" "$PG_CONF"
sudo sed -i "s/^max_connections = .*/max_connections = 20/" "$PG_CONF"
sudo sed -i "s/^work_mem = .*/work_mem = 4MB/" "$PG_CONF"
sudo sed -i "s/^maintenance_work_mem = .*/maintenance_work_mem = 64MB/" "$PG_CONF"

# Allow password auth (for app connection)
sudo sed -i 's/^local\s\+all\s\+all\s\+peer/local   all             all                                     md5/' "$PG_CONF"

sudo systemctl restart postgresql
echo "=== PostgreSQL tuned for 1GB RAM ==="

echo ""
echo "=== Installasi selesai! ==="
echo "PostgreSQL: status=$(systemctl is-active postgresql)"
echo "Redis:      status=$(systemctl is-active redis-server)"
echo ""
echo "Jalankan migrasi: make deploy-migrate"
