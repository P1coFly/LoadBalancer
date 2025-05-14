package httpbackend

import (
	"net"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type backend struct {
	url   *url.URL
	alive bool
	mu    sync.RWMutex
	rp    *httputil.ReverseProxy
}

func NewBackend(rawUrl string) (*backend, error) {
	parsedURL, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	return &backend{
		url:   parsedURL,
		alive: true,
		rp:    httputil.NewSingleHostReverseProxy(parsedURL),
	}, nil
}

func (b *backend) SetAlive(alive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.alive = alive
}

func (b *backend) IsAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.alive
}

func (b *backend) ReverseProxy() *httputil.ReverseProxy {
	return b.rp
}

func (b *backend) CheckHealth(timeout time.Duration) (bool, error) {
	conn, err := net.DialTimeout("tcp", b.url.Host, timeout)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	return true, nil
}

func (b *backend) URLString() string {
	return b.url.Host
}
