package backends

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"

	httpbackend "github.com/P1coFly/LoadBalancer/pkg/backends/http"
	"github.com/P1coFly/LoadBalancer/pkg/handlers"
)

type BackendType string
type contextKey string

const (
	HTTP        BackendType = "HTTP"
	AttemptsKey contextKey  = "attempts"
	MaxRetries  int         = 3
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
	CheckHealth(timeout time.Duration) (bool, error)
	URLString() string
}

type SelectionStrategy interface {
	Next([]Backend) Backend
}

type BackendsPool struct {
	backends []Backend
	strategy SelectionStrategy
	Logger   *slog.Logger
}

func NewPool(strategy SelectionStrategy, bType BackendType, urls []string, logger *slog.Logger) (*BackendsPool, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, "empty URLs list")
	}

	var bs []Backend
	var err error
	bp := &BackendsPool{
		strategy: strategy,
		Logger:   logger,
	}

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
	p.Logger.Error(ErrNoBackends.Error())
	handlers.SendJSONError(w, http.StatusServiceUnavailable, "Service not available")
}

func (p *BackendsPool) HealthCheck(timeout time.Duration) {
	for _, b := range p.backends {
		go func(be Backend) {
			alive, err := be.CheckHealth(timeout)
			if err != nil {
				p.Logger.Error("Backend is not responding", "url", be.URLString(), "error", err)
			}
			be.SetAlive(alive)
			p.Logger.Info("Backend status check", "url", be.URLString(), "alive", alive)
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
			p.Logger.Error("proxy error", "url", b.URLString(), "err", e)

			b.SetAlive(false)
			attempts := GetAttemptsFromContext(req) + 1

			p.Logger.Info("new attemp", "attemps", attempts)
			if attempts < MaxRetries {
				ctx := context.WithValue(req.Context(), AttemptsKey, attempts)
				nextPeer := p.Next()
				if nextPeer != nil {
					nextPeer.ReverseProxy().ServeHTTP(rw, req.WithContext(ctx))
					return
				}
				p.Logger.Error(ErrNoBackends.Error())
				handlers.SendJSONError(rw, http.StatusServiceUnavailable, "Service not available")
				return
			}
			handlers.SendJSONError(rw, http.StatusBadGateway, "Too many retries")
		}

		backends = append(backends, b)
	}
	return backends, nil
}

func GetAttemptsFromContext(r *http.Request) int {
	if v := r.Context().Value(AttemptsKey); v != nil {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}
