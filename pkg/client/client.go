package client

// Структкра описывает бакеты токенов. Используется для реализация Rate-Limiting
type TokenBucket struct {
	Capacity      int
	CurrentTokens int
	RPS           int
}

// Refill добавляет токены, не превышая Capacity
func (tb *TokenBucket) Refill() {
	if tb.Capacity > tb.CurrentTokens+tb.RPS {
		tb.CurrentTokens = tb.CurrentTokens + tb.RPS
		return
	}
	tb.CurrentTokens = tb.Capacity
}

// Allow пытается «потратить» n токенов.
// Возвращает true, если удалось, false — иначе
func (tb *TokenBucket) Allow(n int) bool {
	if tb.CurrentTokens < n {
		return false
	}
	tb.CurrentTokens -= n
	return true
}

// Структкра описывает сущность клиентов
type Client struct {
	ID          string `json:"client_id"`
	TokenBucket TokenBucket
}

func NewClient(id string, capacity, rps int) *Client {
	return &Client{
		ID: id,
		TokenBucket: TokenBucket{
			Capacity:      capacity,
			CurrentTokens: capacity,
			RPS:           rps},
	}
}

type ClientRepo interface {
	GetClient(id string) *Client
	AddClient(id string, capacity, rps int) *Client
	UpdateClient(id string, capacity, rps int) (*Client, error)
	DeleteClient(id string) error

	Consume(id string, n int) bool
	DefaultRPS() int
	DefaultCapacity() int
}
