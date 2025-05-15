package main_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/P1coFly/LoadBalancer/pkg/backends"
	"github.com/P1coFly/LoadBalancer/pkg/backends/strategies"
)

func TestLoadBalancer_DistributesTraffic(t *testing.T) {
	var mu sync.Mutex
	backendHits := make(map[string]int)

	// Создаем два тестовых бэкенда
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		backendHits["backend1"]++
		mu.Unlock()
		if _, err := w.Write([]byte("backend1")); err != nil {
			t.Logf("Error write to responseWriter, err: %t", err)
		}

	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		backendHits["backend2"]++
		mu.Unlock()
		if _, err := w.Write([]byte("backend2")); err != nil {
			t.Logf("Error write to responseWriter, err: %t", err)
		}
	}))
	defer backend2.Close()

	// Инициализируем пул бэкендов с стратегией Round Robin
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	backendURLs := []string{backend1.URL, backend2.URL}
	strategy := strategies.NewRoundRobin()
	pool, err := backends.NewPool(strategy, backends.HTTP, backendURLs, logger)
	if err != nil {
		t.Fatalf("failed to create backend pool: %v", err)
	}

	// Создаем тестовый сервер с балансировщиком
	lbServer := httptest.NewServer(http.HandlerFunc(pool.LoadBalancerHandler))
	defer lbServer.Close()

	// Отправляем 10 запросов к балансировщику
	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 10; i++ {
		resp, err := client.Get(lbServer.URL)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed read response body: %v", err)
		}
		resp.Body.Close()
		if string(body) != "backend1" && string(body) != "backend2" {
			t.Errorf("unexpected response body: %s", body)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if backendHits["backend1"] == 0 || backendHits["backend2"] == 0 {
		t.Errorf("expected both backends to receive traffic, got: %v", backendHits)
	}
}

func TestLoadBalancer_RetryAndRecovery(t *testing.T) {
	var mu sync.Mutex
	hits := map[string]int{"backend1": 0, "backend2": 0}

	// поднимаем оба бэкенда, далее будем управлять здоровьем первым
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits["backend1"]++
		mu.Unlock()
		if _, err := w.Write([]byte("1")); err != nil {
			t.Logf("Error write to responseWriter, err: %t", err)
		}
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits["backend2"]++
		mu.Unlock()
		if _, err := w.Write([]byte("2")); err != nil {
			t.Logf("Error write to responseWriter, err: %t", err)
		}
	}))
	defer srv2.Close()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	pool, err := backends.NewPool(strategies.NewRoundRobin(), backends.HTTP, []string{srv1.URL, srv2.URL}, logger)
	if err != nil {
		t.Fatalf("failed to create backend pool: %v", err)
	}

	lb := httptest.NewServer(http.HandlerFunc(pool.LoadBalancerHandler))
	defer lb.Close()

	client := &http.Client{Timeout: time.Second}

	// 1) Один запрос: пойдет на backend1
	resp, err := client.Get(lb.URL)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed read response body: %v", err)
	}
	resp.Body.Close()
	if string(body) != "1" {
		t.Fatalf("expected from 1, got %s", body)
	}

	// 2) Закрываем первый бэкенд (имитируем падение)
	srv1.CloseClientConnections()
	srv1.Close()

	for i := 0; i < 3; i++ {
		resp, err := client.Get(lb.URL)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed read response body: %v", err)
		}
		resp.Body.Close()
		if string(b) != "2" {
			t.Errorf("expected failover to 2, got %s", b)
		}
	}

	// 3) «Восстанавливаем» первый бэкенд
	srv1 = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits["backend1"]++
		mu.Unlock()
		if _, err := w.Write([]byte("1")); err != nil {
			t.Logf("Error write to responseWriter, err: %t", err)
		}
	}))
	pool, err = backends.NewPool(strategies.NewRoundRobin(), backends.HTTP, []string{srv1.URL, srv2.URL}, logger)
	if err != nil {
		t.Fatalf("failed to create backend pool: %v", err)
	}

	lb.Config.Handler = http.HandlerFunc(pool.LoadBalancerHandler)

	// даем пулу время на «health check»
	time.Sleep(100 * time.Millisecond)

	// 4) Следующий запрос должен вернуться на backend1
	resp, err = client.Get(lb.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed read response body: %v", err)
	}
	resp.Body.Close()
	if string(b) != "1" {
		t.Errorf("expected recovery to 1, got %s", b)
	}

	// проверка счетчиков
	mu.Lock()
	defer mu.Unlock()
	if hits["backend1"] < 2 {
		t.Errorf("backend1 should have >=2 hits, got %d", hits["backend1"])
	}
	if hits["backend2"] < 3 {
		t.Errorf("backend2 should have >=3 hits, got %d", hits["backend2"])
	}
}

// TestLoadBalancer_AllDown проверяет, что когда все бэкенды недоступны,
// балансировщик возвращает 503 Service Unavailable.
func TestLoadBalancer_AllDown(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	pool, err := backends.NewPool(strategies.NewRoundRobin(), backends.HTTP, []string{"123.321.123.311"}, logger)
	if err != nil {
		t.Fatalf("failed to create backend pool: %v", err)
	}

	lb := httptest.NewServer(http.HandlerFunc(pool.LoadBalancerHandler))
	defer lb.Close()

	client := &http.Client{Timeout: time.Second}

	// 1) Один запрос: пойдет на backend1
	resp, err := client.Get(lb.URL)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
	}
}
