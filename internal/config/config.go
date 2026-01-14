package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	MongoDB  MongoDBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Bcrypt   BcryptConfig
}

type ServerConfig struct {
	Port    int    `env:"PORT" envDefault:"8080"`
	GinMode string `env:"GIN_MODE" envDefault:"debug"`
}

type DatabaseConfig struct {
	URL string `env:"DATABASE_URL,required"`
}

type MongoDBConfig struct {
	URI      string `env:"MONGODB_URI,required"`
	Database string `env:"MONGODB_DATABASE" envDefault:"gochat"`
}

type RedisConfig struct {
	URL string `env:"REDIS_URL,required"`
}

type JWTConfig struct {
	Secret        string        `env:"JWT_SECRET,required"`
	AccessExpiry  time.Duration `env:"JWT_ACCESS_EXPIRY" envDefault:"15m"`
	RefreshExpiry time.Duration `env:"JWT_REFRESH_EXPIRY" envDefault:"168h"`
}

type BcryptConfig struct {
	Cost int `env:"BCRYPT_COST" envDefault:"12"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
