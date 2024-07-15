package traceBufferRedis

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	Expire   string `mapstructure:"expire"`
	RedisUrl string `mapstructure:"redis_url"`
	DbName   int    `mapstructure:"db_name"`
	Port     int    `mapstructure:"port"`
	Limit    int    `mapstructure:"limit"`
}

func createDefaultConfig() component.Config {
	return &Config{
		Expire:   "1m",
		RedisUrl: "localhost:6379",
		DbName:   0,
		Port:     8080,
		Limit:    1000,
	}
}

func (c Config) Validate() error {
	if _, err := time.ParseDuration(c.Expire); err != nil {
		return err
	}
	if c.RedisUrl == "" {
		return errors.New("redis_url invalid")
	}
	if c.Port < 0 || 65535 < c.Port {
		return errors.New("port invalid")
	}
	if c.Limit <= 0 {
		return errors.New("limit invalid")
	}

	return nil
}
