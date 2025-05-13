package strategies

import (
	"sync/atomic"

	"github.com/P1coFly/LoadBalancer/pkg/backends"
)

type RoundRobinStrategy struct {
	current uint64
}

func NewRoundRobin() *RoundRobinStrategy {
	return &RoundRobinStrategy{current: 0}
}

func (s *RoundRobinStrategy) Next(backends []backends.Backend) backends.Backend {
	countBackends := len(backends)
	if countBackends == 0 {
		return nil
	}

	start := atomic.AddUint64(&s.current, 1)
	start--

	for i := 0; i < countBackends; i++ {
		index := (start + uint64(i)) % uint64(countBackends)
		backend := backends[index]
		if backend.IsAlive() {
			if i > 0 {
				atomic.StoreUint64(&s.current, index)
			}
			return backend
		}
	}
	return nil
}
