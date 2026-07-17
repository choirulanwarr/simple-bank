package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoad_WithEnvFile(t *testing.T) {
	// Create a temporary .env file
	envContent := `
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=testuser
POSTGRES_PASSWORD=testpass
POSTGRES_DB=testdb
GRPC_SERVER_ADDRESS=0.0.0.0:9090
TOKEN_SYMMETRIC_KEY=12345678901234567890123456789012
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=24h
REDIS_HOST=localhost
REDIS_PORT=6379
`
	err := os.WriteFile(".env.test", []byte(envContent), 0644)
	require.NoError(t, err)
	defer os.Remove(".env.test")

	// Temporarily rename .env.test to .env for viper to find it
	err = os.Rename(".env.test", ".env")
	require.NoError(t, err)
	defer os.Remove(".env")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "localhost", cfg.DBHost)
	require.Equal(t, 5432, cfg.DBPort)
	require.Equal(t, "testuser", cfg.DBUser)
	require.Equal(t, "testpass", cfg.DBPassword)
	require.Equal(t, "testdb", cfg.DBName)
	require.Equal(t, "0.0.0.0:9090", cfg.GRPCServerAddress)
	require.Equal(t, "12345678901234567890123456789012", cfg.TokenSymmetricKey)
	require.Equal(t, 15*time.Minute, cfg.AccessTokenDuration)
	require.Equal(t, 24*time.Hour, cfg.RefreshTokenDuration)
	require.Equal(t, "localhost", cfg.RedisHost)
	require.Equal(t, 6379, cfg.RedisPort)
}

func TestLoad_WithoutEnvFile(t *testing.T) {
	// Ensure no .env file exists
	os.Remove(".env")

	// Set environment variables
	os.Setenv("POSTGRES_HOST", "env-host")
	os.Setenv("POSTGRES_PORT", "5433")
	os.Setenv("POSTGRES_USER", "envuser")
	os.Setenv("POSTGRES_PASSWORD", "envpass")
	os.Setenv("POSTGRES_DB", "envdb")
	os.Setenv("GRPC_SERVER_ADDRESS", "0.0.0.0:8080")
	os.Setenv("TOKEN_SYMMETRIC_KEY", "abcdefghijklmnopqrstuvwxyz123456")
	os.Setenv("ACCESS_TOKEN_DURATION", "30m")
	os.Setenv("REFRESH_TOKEN_DURATION", "48h")
	os.Setenv("REDIS_HOST", "env-redis")
	os.Setenv("REDIS_PORT", "6380")

	defer func() {
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB")
		os.Unsetenv("GRPC_SERVER_ADDRESS")
		os.Unsetenv("TOKEN_SYMMETRIC_KEY")
		os.Unsetenv("ACCESS_TOKEN_DURATION")
		os.Unsetenv("REFRESH_TOKEN_DURATION")
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
	}()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "env-host", cfg.DBHost)
	require.Equal(t, 5433, cfg.DBPort)
	require.Equal(t, "envuser", cfg.DBUser)
	require.Equal(t, "envpass", cfg.DBPassword)
	require.Equal(t, "envdb", cfg.DBName)
	require.Equal(t, "0.0.0.0:8080", cfg.GRPCServerAddress)
	require.Equal(t, "abcdefghijklmnopqrstuvwxyz123456", cfg.TokenSymmetricKey)
	require.Equal(t, 30*time.Minute, cfg.AccessTokenDuration)
	require.Equal(t, 48*time.Hour, cfg.RefreshTokenDuration)
	require.Equal(t, "env-redis", cfg.RedisHost)
	require.Equal(t, 6380, cfg.RedisPort)
}

func TestDatabaseURL(t *testing.T) {
	cfg := &Config{
		DBUser:     "user",
		DBPassword: "pass",
		DBHost:     "localhost",
		DBPort:     5432,
		DBName:     "dbname",
	}

	url := cfg.DatabaseURL()
	expected := "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
	require.Equal(t, expected, url)
}