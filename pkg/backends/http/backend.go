package httpbackend

import (
	"log"
	"net"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type backend struct {
	url   *url.URL
	alive bool
	mux   sync.RWMutex
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
	b.mux.Lock()
	defer b.mux.Unlock()
	b.alive = alive
}

func (b *backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.alive
}

func (b *backend) ReverseProxy() *httputil.ReverseProxy {
	return b.rp
}

func (b *backend) CheckHealth(timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", b.url.Host, timeout)
	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	defer conn.Close()
	return true
}

func (b *backend) URLString() string {
	return b.url.Host
}
