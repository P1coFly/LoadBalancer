package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"log/slog"

	"github.com/P1coFly/LoadBalancer/internal/config"
	"github.com/P1coFly/LoadBalancer/pkg/client"
	"github.com/P1coFly/LoadBalancer/pkg/handlers"
	"github.com/P1coFly/LoadBalancer/pkg/middleware"
)

type clientResponse struct {
	ClientID      string `json:"client_id"`
	Capacity      int    `json:"capacity"`
	CurrentTokens int    `json:"current_tokens"`
	RPS           int    `json:"rate_per_sec"`
}

const (
	Port = ":8080"
)

// Запускает сервер на выбранном порту
func startTestServer(t *testing.T) (addr string, shutdown func()) {
	// минимальный config
	cfg := &config.Config{
		Env: "dev",
		Server: config.Server{
			Port:           Port,
			ReadTimeout:    2 * time.Second,
			WriteTimeout:   2 * time.Second,
			IdleTimeout:    2 * time.Second,
			HealthInterval: 100 * time.Millisecond,
			Backends:       []string{"http://invalid"},
		},
		RateLimit: config.RateLimit{
			DefaultCapacity:   10,
			DefaultRPS:        5,
			ReplenishInterval: 100 * time.Millisecond,
		},
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// собираем mux из main()
	clientRepo := client.NewMemoryRepo(cfg.RateLimit.DefaultCapacity, cfg.RateLimit.DefaultRPS, logger)
	// запускаем replenish
	stopRepl := make(chan struct{})
	go func() {
		ticker := time.NewTicker(cfg.RateLimit.ReplenishInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				clientRepo.Replenish()
			case <-stopRepl:
				return
			}
		}
	}()

	// handlers
	ch := &handlers.ClientHandler{Repo: clientRepo, Logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ch.Create(w, r)
		case http.MethodGet:
			ch.Get(w, r)
		case http.MethodPut:
			ch.Update(w, r)
		case http.MethodDelete:
			ch.Delete(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// балансировщик не нужен в интеграции клиентского CRUD

	srv := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      middleware.AccessLog(logger, mux),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// слушаем на случайном порту
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr = ln.Addr().String()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Serve(ln); err != nil {
			t.Logf("Error to start srv, err: %t", err)
		}
	}()

	shutdown = func() {
		close(stopRepl)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Logf("Error to shutdown srv, err: %t", err)
		}
		wg.Wait()
	}
	return addr, shutdown
}

func TestIntegration_ClientCRUD(t *testing.T) {
	addr, shutdown := startTestServer(t)
	defer shutdown()

	clientURL := "http://" + addr + "/clients"

	// CREATE
	createBody := `{"client_id":"u1","capacity":3,"rate_per_sec":1}`
	res, err := http.Post(clientURL, "application/json", bytes.NewBufferString(createBody))
	if err != nil {
		t.Fatalf("POST error: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var cr clientResponse
	if err := json.NewDecoder(res.Body).Decode(&cr); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if cr.ClientID != "u1" || cr.Capacity != 3 || cr.CurrentTokens != 3 {
		t.Errorf("unexpected create resp: %+v", cr)
	}

	// GET
	res, err = http.Get(clientURL + "?client_id=u1")
	if err != nil {
		t.Fatalf("GET error: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 GET, got %d", res.StatusCode)
	}
	var gr clientResponse
	if err := json.NewDecoder(res.Body).Decode(&gr); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if gr.ClientID != "u1" {
		t.Errorf("unexpected get resp: %+v", gr)
	}

	// UPDATE
	updateBody := `{"client_id":"u1","capacity":5,"rate_per_sec":2}`
	req, _ := http.NewRequest(http.MethodPut, clientURL, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT error: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 PUT, got %d", res.StatusCode)
	}
	var ur clientResponse
	if err := json.NewDecoder(res.Body).Decode(&ur); err != nil {
		t.Fatalf("decode put: %v", err)
	}
	if ur.Capacity != 5 || ur.CurrentTokens != 5 || ur.RPS != 2 {
		t.Errorf("unexpected put resp: %+v", ur)
	}

	// DELETE
	req, _ = http.NewRequest(http.MethodDelete, clientURL+"?client_id=u1", nil)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE error: %v", err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 DELETE, got %d", res.StatusCode)
	}

	res, err = http.Get(clientURL + "?client_id=u1")
	if err != nil {
		t.Fatalf("GET after delete error: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", res.StatusCode)
	}
}
