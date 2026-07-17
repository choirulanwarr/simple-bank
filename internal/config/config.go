package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DBHost             string        `mapstructure:"POSTGRES_HOST"`
	DBPort             int           `mapstructure:"POSTGRES_PORT"`
	DBUser             string        `mapstructure:"POSTGRES_USER"`
	DBPassword         string        `mapstructure:"POSTGRES_PASSWORD"`
	DBName             string        `mapstructure:"POSTGRES_DB"`
	GRPCServerAddress  string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	TokenSymmetricKey  string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	RedisHost          string        `mapstructure:"REDIS_HOST"`
	RedisPort          int           `mapstructure:"REDIS_PORT"`
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// Check if it's a "file not found" error (either viper.ConfigFileNotFoundError or os.PathError)
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Check for os.PathError (when .env file doesn't exist)
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) {
				return nil, err
			}
		}
		// .env file not found is OK, continue with env vars only
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}