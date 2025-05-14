package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config описывает все параметры приложения
type Config struct {
	Env string `yaml:"env" env-required:"true"`

	Server    Server    `yaml:"server"`
	RateLimit RateLimit `yaml:"rate_limit"`
	Features  Features  `yaml:"features"`
}

// Server содержит настройки HTTP-сервера
type Server struct {
	Port           string        `yaml:"port" env-required:"true"`
	ReadTimeout    time.Duration `yaml:"timeouts.read" env-default:"10s"`
	WriteTimeout   time.Duration `yaml:"timeouts.write" env-default:"10s"`
	IdleTimeout    time.Duration `yaml:"timeouts.idle" env-default:"60s"`
	HealthInterval time.Duration `yaml:"health_interval" env-default:"30s"`
	Backends       []string      `yaml:"backends" env-required:"true"`
}

// RateLimit содержит параметры Token Bucket
type RateLimit struct {
	DefaultCapacity   int           `yaml:"default_capacity" env-default:"10"`
	DefaultRPS        int           `yaml:"default_rps" env-default:"1"`
	ReplenishInterval time.Duration `yaml:"replenish_interval" env-default:"1s"`
}

// Features флаги включения отдельных модулей
type Features struct {
	HealthCheck bool `yaml:"healthcheck" env-default:"true"`
	RateLimit   bool `yaml:"rate_limit" env-default:"true"`
}

// MustLoad читает конфиг из файла, ENV, флагов и проводит валидацию
func MustLoad() *Config {
	var cfg Config

	// Флаг для пути к файлу конфига
	path := flag.String("config", os.Getenv("CONFIG_PATH"), "path to config file")
	flag.Parse()

	if *path == "" {
		log.Fatal("config path must be set via --config or CONFIG_PATH")
	}

	// Читаем из YAML
	if err := cleanenv.ReadConfig(*path, &cfg); err != nil {
		log.Fatalf("ReadConfig failed: %v", err)
	}

	// Валидация обязательных полей
	if cfg.Env == "" {
		log.Fatal("env is required")
	}
	if len(cfg.Server.Backends) == 0 {
		log.Fatal("at least one backend required")
	}

	return &cfg
}
