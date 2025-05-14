package client

type Client struct {
	ID       string `json:"client_id"`
	Capacity int    `json:"capacity"`
	RPS      int    `json:"rate_per_sec"`
}

func NewClient(id string, capacity, rps int) *Client {
	return &Client{
		ID:       id,
		Capacity: capacity,
		RPS:      rps,
	}
}

type ClientRepo interface {
	GetClient(id string) *Client
	AddClient(id string, capacity, rps int) *Client
	UpdateClient(id string, capacity, rps int) (*Client, error)
	DeleteClient(id string) error

	DefaultRPS() int
	DefaultCapacity() int
}
