package config

import (
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

// MustLoad читает конфиг из файла и проводит валидацию. Путь до конфига берёт из переменой окружения CONFIG_PATH
func MustLoad() *Config {
	var cfg Config

	configPath := os.Getenv("CONFIG_PATH")
	log.Printf("%s", configPath)
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("can't read config: %s", err)
	}

	return &cfg
}
