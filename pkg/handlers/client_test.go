package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/P1coFly/LoadBalancer/pkg/client"
)

// helper создаёт новый хендлер с свежим репо и тест‑логгер
func newTestHandler() (*ClientHandler, *client.ClientMemoryRepository) {
	repo := client.NewMemoryRepo(10, 2, slog.New(slog.NewTextHandler(nil, nil)))
	h := &ClientHandler{
		Repo:   repo,
		Logger: slog.New(slog.NewTextHandler(nil, nil)),
	}
	return h, repo
}

func TestCreate_Success(t *testing.T) {
	h, _ := newTestHandler()
	body := `{"client_id":"u1","capacity":5,"rate_per_sec":1}`
	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("want %d, got %d", http.StatusCreated, rr.Code)
	}
	var resp clientResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ClientID != "u1" || resp.Capacity != 5 || resp.RPS != 1 || resp.CurrentTokens != 5 {
		t.Errorf("unexpected resp: %+v", resp)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewBufferString(`{`))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreate_MissingID(t *testing.T) {
	h, _ := newTestHandler()
	body := `{"capacity":5,"rate_per_sec":1}`
	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreate_Conflict(t *testing.T) {
	h, repo := newTestHandler()
	// заранее создаём клиента
	repo.AddClient("u1", 3, 1)

	body := `{"client_id":"u1","capacity":5,"rate_per_sec":1}`
	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("want %d, got %d", http.StatusConflict, rr.Code)
	}
}

func TestGet_Success(t *testing.T) {
	h, repo := newTestHandler()
	repo.AddClient("u2", 7, 2)

	req := httptest.NewRequest(http.MethodGet, "/clients?client_id=u2", nil)
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, rr.Code)
	}
	var resp clientResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ClientID != "u2" || resp.Capacity != 7 {
		t.Errorf("unexpected resp: %+v", resp)
	}
}

func TestGet_MissingID(t *testing.T) {
	h, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/clients", nil)
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGet_NotFound(t *testing.T) {
	h, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/clients?client_id=unknown", nil)
	rr := httptest.NewRecorder()

	h.Get(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("want %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestUpdate_Success(t *testing.T) {
	h, repo := newTestHandler()
	repo.AddClient("u3", 4, 1)

	body := `{"client_id":"u3","capacity":10,"rate_per_sec":5}`
	req := httptest.NewRequest(http.MethodPut, "/clients", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, rr.Code)
	}
	var resp clientResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Capacity != 10 || resp.CurrentTokens != 10 || resp.RPS != 5 {
		t.Errorf("unexpected resp: %+v", resp)
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	h, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodPut, "/clients", bytes.NewBufferString(`{`))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestUpdate_MissingID(t *testing.T) {
	h, _ := newTestHandler()
	body := `{"capacity":5,"rate_per_sec":1}`
	req := httptest.NewRequest(http.MethodPut, "/clients", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	h, _ := newTestHandler()
	body := `{"client_id":"nope","capacity":5,"rate_per_sec":1}`
	req := httptest.NewRequest(http.MethodPut, "/clients", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("want %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestDelete_Success(t *testing.T) {
	h, repo := newTestHandler()
	repo.AddClient("u4", 3, 1)

	req := httptest.NewRequest(http.MethodDelete, "/clients?client_id=u4", nil)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("want %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestDelete_MissingID(t *testing.T) {
	h, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodDelete, "/clients", nil)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("want %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	h, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodDelete, "/clients?client_id=none", nil)
	rr := httptest.NewRecorder()

	h.Delete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("want %d, got %d", http.StatusNotFound, rr.Code)
	}
}
