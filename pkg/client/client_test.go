package client_test

import (
	"testing"

	"log/slog"
	"os"

	"github.com/P1coFly/LoadBalancer/pkg/client"
)

func setupRepo() *client.ClientMemoryRepository {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return client.NewMemoryRepo(10, 5, logger)
}

func TestAddAndGetClient(t *testing.T) {
	repo := setupRepo()
	id := "test-client"
	repo.AddClient(id, 20, 10)

	cl := repo.GetClient(id)
	if cl == nil {
		t.Fatal("expected client to be added")
	}
	if cl.TokenBucket.Capacity != 20 || cl.TokenBucket.RPS != 10 {
		t.Errorf("unexpected token bucket values: %+v", cl.TokenBucket)
	}
}

func TestUpdateClient(t *testing.T) {
	repo := setupRepo()
	id := "client-update"
	repo.AddClient(id, 10, 5)

	updated, err := repo.UpdateClient(id, 30, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.TokenBucket.Capacity != 30 || updated.TokenBucket.RPS != 15 {
		t.Errorf("update failed: %+v", updated.TokenBucket)
	}
}

func TestDeleteClient(t *testing.T) {
	repo := setupRepo()
	id := "client-delete"
	repo.AddClient(id, 10, 5)

	err := repo.DeleteClient(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.GetClient(id) != nil {
		t.Error("client should be deleted")
	}
}

func TestConsumeTokens(t *testing.T) {
	repo := setupRepo()
	id := "client-consume"
	repo.AddClient(id, 5, 2)

	ok := repo.Consume(id, 3)
	if !ok {
		t.Error("expected to consume tokens")
	}
	cl := repo.GetClient(id)
	if cl.TokenBucket.CurrentTokens != 2 {
		t.Errorf("unexpected token count: %d", cl.TokenBucket.CurrentTokens)
	}
}

func TestReplenishTokens(t *testing.T) {
	repo := setupRepo()
	id := "client-replenish"
	repo.AddClient(id, 10, 3)
	repo.Consume(id, 5)

	repo.Replenish()
	cl := repo.GetClient(id)
	if cl.TokenBucket.CurrentTokens != 8 {
		t.Errorf("expected 8 tokens, got %d", cl.TokenBucket.CurrentTokens)
	}
}
