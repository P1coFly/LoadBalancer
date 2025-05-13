package backends

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	httpbackend "github.com/P1coFly/LoadBalancer/pkg/backends/http"
)

type BackendType string
type contextKey string

const (
	HTTP BackendType = "HTTP"
	// RetryKey    contextKey  = "retry"
	AttemptsKey contextKey = "attempts"
	MaxRetries  int        = 3
)

var (
	ErrWrongType    = errors.New("unsupported backend type")
	ErrNoBackends   = errors.New("at least one backend required")
	ErrInvalidInput = errors.New("invalid input parameters")
)

type Backend interface {
	IsAlive() bool
	SetAlive(bool)
	ReverseProxy() *httputil.ReverseProxy
	CheckHealth(timeout time.Duration) bool
	URLString() string
}

type SelectionStrategy interface {
	Next([]Backend) Backend
}

type BackendsPool struct {
	backends []Backend
	strategy SelectionStrategy
}

func NewPool(strategy SelectionStrategy, bType BackendType, urls []string) (*BackendsPool, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, "empty URLs list")
	}

	var bs []Backend
	var err error
	bp := &BackendsPool{strategy: strategy}
	switch bType {
	case HTTP:
		bs, err = createHTTPBackends(urls, bp)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP backends: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w: %s", ErrWrongType, bType)
	}

	bp.backends = bs
	return bp, nil
}

func (p *BackendsPool) Next() Backend {
	return p.strategy.Next(p.backends)
}

func (p *BackendsPool) LoadBalancerHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), AttemptsKey, 0)
	r = r.WithContext(ctx)

	peer := p.Next()
	if peer != nil {
		rp := peer.ReverseProxy()
		rp.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// Дополнительные методы
func (p *BackendsPool) HealthCheck(timeout time.Duration) {
	for _, b := range p.backends {
		go func(be Backend) {
			alive := be.CheckHealth(timeout)
			be.SetAlive(alive)
		}(b)
	}
}

func createHTTPBackends(urls []string, p *BackendsPool) ([]Backend, error) {
	backends := make([]Backend, 0, len(urls))
	for _, u := range urls {
		b, err := httpbackend.NewBackend(u)
		if err != nil {
			return nil, fmt.Errorf("invalid URL %q: %w", u, err)
		}
		b.ReverseProxy().ErrorHandler = func(rw http.ResponseWriter, req *http.Request, e error) {
			log.Printf("proxy error for %s: %v\n", b.URLString(), e)

			b.SetAlive(false)
			attempts := GetAttemptsFromContext(req)

			log.Printf("attemps %v", attempts)
			if attempts < MaxRetries {
				ctx := context.WithValue(req.Context(), AttemptsKey, attempts+1)
				nextPeer := p.Next()
				nextPeer.ReverseProxy().ServeHTTP(rw, req.WithContext(ctx))
				return
			}
			http.Error(rw, "Too many retries", http.StatusBadGateway)
		}

		backends = append(backends, b)
	}
	return backends, nil
}

// func GetRetryFromContext(r *http.Request) int {
// 	if v := r.Context().Value(RetryKey); v != nil {
// 		if i, ok := v.(int); ok {
// 			return i
// 		}
// 	}
// 	return 0
// }

func GetAttemptsFromContext(r *http.Request) int {
	if v := r.Context().Value(AttemptsKey); v != nil {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}
