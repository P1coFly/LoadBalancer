package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env    string `yaml:"env"`
	Server `yaml:"server"`
}

type Server struct {
	Port     string   `yaml:"port" env-default:"localhost:8080"`
	Backends []string `yaml:"backends"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	log.Printf("%s", configPath)
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("can't read config: %s", err)
	}

	return &cfg

}
