package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/P1coFly/LoadBalancer/internal/config"
	"github.com/P1coFly/LoadBalancer/pkg/backends"
	"github.com/P1coFly/LoadBalancer/pkg/backends/strategies"
)

func main() {
	// читаем конфиг
	cfg := config.MustLoad()
	// инициализируем логер
	log := setupLogger(cfg.Env)

	log.Info("starting api-servies", "env", cfg.Env)
	log.Debug("cfg data", "data", cfg)

	log.Info("starting server", slog.String("port", cfg.Port))

	strat := strategies.NewRoundRobin()

	backendsPool, err := backends.NewPool(strat, "HTTP", cfg.Backends)
	if err != nil {
		log.Error("failed to create backends pool", "error", err)
		os.Exit(1)
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			backendsPool.HealthCheck(2 * time.Second)
		}
	}()

	// инициализируем server и запускаем
	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      http.HandlerFunc(backendsPool.LoadBalancerHandler),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server", "error", err)
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case "dev":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "prod":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log

}
