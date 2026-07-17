package config

import (
	"fmt"
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
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
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