package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	HTTP      HTTPConfig      `envPrefix:"HTTP_"`
	MySQL     MySQLConfig     `envPrefix:"MYSQL_"`
	Redis     RedisConfig     `envPrefix:"REDIS_"`
	JWT       JwtConfig       `envPrefix:"JWT_"`
	RateLimit RateLimitConfig `envPrefix:"RATE_LIMIT_"`
}

type HTTPConfig struct {
	Port string `env:"PORT" envDefault:"8080"`
}

type MySQLConfig struct {
	Host     string `env:"HOST,required,notEmpty"`
	Port     string `env:"PORT" envDefault:"3306"`
	User     string `env:"USER,required,notEmpty"`
	Password string `env:"PASSWORD,required,notEmpty"`
	Database string `env:"DATABASE,required,notEmpty"`
}

type RedisConfig struct {
	Host string `env:"HOST,required,notEmpty"`
	Port string `env:"PORT" envDefault:"6379"`
}

type JwtConfig struct {
	Secret     string `env:"SECRET,required,notEmpty"`
	TTLMinutes int    `env:"TTL_MINUTES" envDefault:"60"`
}

type RateLimitConfig struct {
	RPS   float64 `env:"RPS" envDefault:"10"`
	Burst int     `env:"BURST" envDefault:"20"`
}

func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.RateLimit.RPS <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_RPS must be greater than 0")
	}

	if cfg.RateLimit.Burst <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_BURST must be greater than 0")
	}

	return cfg, nil
}
