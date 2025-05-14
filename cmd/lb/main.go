package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/P1coFly/LoadBalancer/internal/config"
	"github.com/P1coFly/LoadBalancer/pkg/backends"
	"github.com/P1coFly/LoadBalancer/pkg/backends/strategies"
	"github.com/P1coFly/LoadBalancer/pkg/client"
	"github.com/P1coFly/LoadBalancer/pkg/middleware"
)

func main() {
	// читаем конфиг
	cfg := config.MustLoad()
	// инициализируем логер
	log := setupLogger(cfg.Env)

	log.Info("starting loud balancer", "env", cfg.Env)
	log.Debug("cfg data", "data", cfg)

	strat := strategies.NewRoundRobin()

	backendsPool, err := backends.NewPool(strat, "HTTP", cfg.Server.Backends, log)
	if err != nil {
		log.Error("failed to create backends pool", "error", err)
		os.Exit(1)
	}

	clientRepo := client.NewMemoryRepo(3, 1, log)
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			clientRepo.Replenish()
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.Server.HealthInterval)
		for range ticker.C {
			backendsPool.HealthCheck(2 * time.Second)
		}
	}()

	handlerFunc := http.HandlerFunc(backendsPool.LoadBalancerHandler)
	handler := middleware.AccessLog(log, handlerFunc)
	handler = middleware.RateLimitMiddleware(clientRepo, log, handler)
	// инициализируем server и запускаем
	srv := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	// Запускаем HTTP‑сервер в горутине
	go func() {
		log.Info("server starting", "addr", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	gracefulShutdown(srv, log, 15*time.Second, stop)
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

func gracefulShutdown(srv *http.Server, logger *slog.Logger, timeout time.Duration, stopCh <-chan os.Signal) {
	sig := <-stopCh
	logger.Info("received signal, shutting down", "signal", sig)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		_ = srv.Close()
	}
	logger.Info("server stopped")
}
