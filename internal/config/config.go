package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DBHost               string        `mapstructure:"POSTGRES_HOST"`
	DBPort               int           `mapstructure:"POSTGRES_PORT"`
	DBUser               string        `mapstructure:"POSTGRES_USER"`
	DBPassword           string        `mapstructure:"POSTGRES_PASSWORD"`
	DBName               string        `mapstructure:"POSTGRES_DB"`
	GRPCServerAddress    string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	RedisHost            string        `mapstructure:"REDIS_HOST"`
	RedisPort            int           `mapstructure:"REDIS_PORT"`
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Baca .env — optional, fallback ke env vars
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) {
				return nil, err
			}
		}
	}

	// Baca satu-per-satu via Get() — AutomaticEnv() cuma jalan lewat Get(), bukan AllSettings/Unmarshal
	cfg := Config{
		DBHost:               viper.GetString("POSTGRES_HOST"),
		DBPort:               viper.GetInt("POSTGRES_PORT"),
		DBUser:               viper.GetString("POSTGRES_USER"),
		DBPassword:           viper.GetString("POSTGRES_PASSWORD"),
		DBName:               viper.GetString("POSTGRES_DB"),
		GRPCServerAddress:    viper.GetString("GRPC_SERVER_ADDRESS"),
		TokenSymmetricKey:    viper.GetString("TOKEN_SYMMETRIC_KEY"),
		AccessTokenDuration:  viper.GetDuration("ACCESS_TOKEN_DURATION"),
		RefreshTokenDuration: viper.GetDuration("REFRESH_TOKEN_DURATION"),
		RedisHost:            viper.GetString("REDIS_HOST"),
		RedisPort:            viper.GetInt("REDIS_PORT"),
	}
	return &cfg, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}
