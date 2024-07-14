package traceBufferRedis

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	expire   string `mapstructure:"expire"`
	redisUrl string `mapstructure:"redis_url"`
	dbName   int    `mapstructure:"db_name"`
	port     int    `mapstructure:"port"`
	limit    int    `mapstructure:"limit"`
}

func createDefaultConfig() component.Config {
	return &Config{
		expire:   "1m",
		redisUrl: "localhost:6379",
		dbName:   0,
		port:     8080,
		limit:    1000,
	}
}

func (c Config) Validate() error {
	if _, err := time.ParseDuration(c.expire); err != nil {
		return err
	}
	if c.redisUrl == "" {
		return errors.New("redis_url invalid")
	}
	if c.port < 0 || 65535 < c.port {
		return errors.New("port invalid")
	}
	if c.limit < 0 {
		return errors.New("limit invalid")
	}

	return nil
}
