package client

import (
	"errors"
	"log/slog"
	"sync"
)

var (
	ErrNoClient = errors.New("Client no found")
)

// Репозиторий клиентов. Реализовывает интерфейс ClientRepo
type ClientMemoryRepository struct {
	clients         map[string]*Client
	mu              sync.RWMutex
	defaultCapacity int
	defaultRPS      int
	logger          *slog.Logger
}

func NewMemoryRepo(defaultCapacity, defaultRPS int, logger *slog.Logger) *ClientMemoryRepository {
	return &ClientMemoryRepository{
		clients:         make(map[string]*Client),
		defaultCapacity: defaultCapacity,
		defaultRPS:      defaultRPS,
		logger:          logger,
	}
}

func (c *ClientMemoryRepository) AddClient(id string, capacity, rps int) *Client {
	client := NewClient(id, capacity, rps)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients[id] = client
	c.logger.Debug("Created new client", "id", client.ID, "capacity", client.TokenBucket.Capacity, "rps", client.TokenBucket.RPS)
	return client
}

func (c *ClientMemoryRepository) GetClient(id string) *Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.logger.Debug("client.GetClient", "client id", id)
	return c.clients[id]
}

func (c *ClientMemoryRepository) DeleteClient(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger.Debug("DeleteClient", "id", id)
	if _, ok := c.clients[id]; !ok {
		return ErrNoClient
	}
	delete(c.clients, id)
	return nil
}

func (c *ClientMemoryRepository) UpdateClient(id string, capacity, rps int) (*Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cl, ok := c.clients[id]
	if !ok {
		return nil, ErrNoClient
	}
	cl.TokenBucket.Capacity = capacity
	cl.TokenBucket.CurrentTokens = capacity
	cl.TokenBucket.RPS = rps
	c.logger.Debug("UpdateClient", "id", id, "capacity", capacity, "rps", rps)
	return cl, nil
}

func (c *ClientMemoryRepository) DefaultRPS() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.defaultRPS
}

func (c *ClientMemoryRepository) DefaultCapacity() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.defaultCapacity
}

func (c *ClientMemoryRepository) Replenish() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cl := range c.clients {
		cl.TokenBucket.Refill()
	}
	c.logger.Debug("Replenished capacities for all clients")
}

// getOrCreate возвращает существующего или создаёт нового клиента
func (c *ClientMemoryRepository) getOrCreate(id string) *Client {
	cl, ok := c.clients[id]
	if !ok {
		cl = NewClient(id, c.defaultCapacity, c.defaultRPS)
		c.clients[id] = cl
	}
	return cl
}

// Consume — пытаемся потратить токены, если клиента нет, то создаст нового с дефолтными параметрами
func (c *ClientMemoryRepository) Consume(id string, n int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	cl := c.getOrCreate(id)
	return cl.TokenBucket.Allow(n)
}
