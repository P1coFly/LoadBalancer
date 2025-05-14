package client

import (
	"errors"
	"log/slog"
	"sync"
)

var (
	ErrNoClient = errors.New("Client no found")
)

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
	c.logger.Info("Created new client", "id", client.ID, "capacity", client.Capacity, "rps", client.RPS)
	return client
}

func (c *ClientMemoryRepository) GetClient(id string) *Client {
	c.logger.Info("client.GetClient", "client id", id)
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.clients[id]
}

func (c *ClientMemoryRepository) DeleteClient(id string) error {
	c.logger.Info("DeleteClient", "id", id)
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.clients[id]; !ok {
		return ErrNoClient
	}
	delete(c.clients, id)
	return nil
}

func (c *ClientMemoryRepository) UpdateClient(id string, capacity, rps int) (*Client, error) {
	c.logger.Info("UpdateClient", "id", id, "capacity", capacity, "rps", rps)
	c.mu.Lock()
	defer c.mu.Unlock()
	cl, ok := c.clients[id]
	if !ok {
		return nil, ErrNoClient
	}
	cl.Capacity = capacity
	cl.RPS = rps
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
	c.logger.Debug("Start replenish capacities for all clients")
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cl := range c.clients {
		cl.Capacity += cl.RPS
	}
	c.logger.Debug("Replenished capacities for all clients")
}
