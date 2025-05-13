// file: pkg/backends/strategies/round_robin_test.go
package strategies

import (
	"net/http/httputil"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/P1coFly/LoadBalancer/pkg/backends"
)

// fakeBackend — Mock реализация backends.Backend
type fakeBackend struct {
	alive      bool
	identifier string
}

func (f *fakeBackend) IsAlive() bool {
	return f.alive
}
func (f *fakeBackend) SetAlive(a bool) {
	f.alive = a
}
func (f *fakeBackend) ReverseProxy() *httputil.ReverseProxy {
	return nil
}
func (f *fakeBackend) CheckHealth(timeout time.Duration) bool {
	return f.alive
}
func (f *fakeBackend) URLString() string {
	return f.identifier
}

func TestNext_Empty(t *testing.T) {
	strat := NewRoundRobin()
	if got := strat.Next(nil); got != nil {
		t.Errorf("Next(nil) = %v; want nil", got)
	}
	if got := strat.Next([]backends.Backend{}); got != nil {
		t.Errorf("Next(empty) = %v; want nil", got)
	}
}

func TestNext_AllAlive_Circle(t *testing.T) {
	strat := NewRoundRobin()
	bs := []backends.Backend{
		&fakeBackend{alive: true, identifier: "A"},
		&fakeBackend{alive: true, identifier: "B"},
		&fakeBackend{alive: true, identifier: "C"},
	}

	want := []string{"A", "B", "C", "A", "B"}
	for i, exp := range want {
		fb := strat.Next(bs)
		if fb == nil || fb.URLString() != exp {
			t.Errorf("iteration %d: got %v; want %s", i, fb.URLString(), exp)
		}
	}
}

func TestNext_SkipDead(t *testing.T) {
	strat := NewRoundRobin()
	bs := []backends.Backend{
		&fakeBackend{alive: true, identifier: "A"},
		&fakeBackend{alive: false, identifier: "B"},
		&fakeBackend{alive: true, identifier: "C"},
	}

	want := []string{"A", "C"}
	for i, exp := range want {
		fb := strat.Next(bs)
		if fb == nil || fb.URLString() != exp {
			t.Errorf("iteration %d: got %v; want %s", i, fb, exp)
		}
	}
}

func TestNext_AllDead(t *testing.T) {
	strat := NewRoundRobin()
	bs := []backends.Backend{
		&fakeBackend{alive: false, identifier: "X"},
		&fakeBackend{alive: false, identifier: "Y"},
	}

	if got := strat.Next(bs); got != nil {
		t.Errorf("Next(all dead) = %v; want nil", got)
	}
}

func TestConcurrentSafety(t *testing.T) {
	strat := NewRoundRobin()
	bs := []backends.Backend{
		&fakeBackend{alive: true, identifier: "A"},
		&fakeBackend{alive: true, identifier: "B"},
	}

	var countA, countB int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fb := strat.Next(bs)
			if fb.URLString() == "A" {
				atomic.AddInt32(&countA, 1)
			} else if fb.URLString() == "B" {
				atomic.AddInt32(&countB, 1)
			}
		}()
	}
	wg.Wait()

	if countA == 0 || countB == 0 {
		t.Errorf("concurrent Next: countA=%d, countB=%d; want >0 each", countA, countB)
	}
}
