package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/P1coFly/LoadBalancer/pkg/client"
)

type clientRequest struct {
	ClientID string `json:"client_id"`
	Capacity int    `json:"capacity"`
	RPS      int    `json:"rate_per_sec"`
}

type clientResponse struct {
	ClientID      string `json:"client_id"`
	Capacity      int    `json:"capacity"`
	CurrentTokens int    `json:"current_tokens"`
	RPS           int    `json:"rate_per_sec"`
}

// ClientHandler хранит репо и логгер
type ClientHandler struct {
	Repo   client.ClientRepo
	Logger *slog.Logger
}

// POST /clients
func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req clientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.ClientID == "" {
		SendJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	cl := h.Repo.GetClient(req.ClientID)
	if cl != nil {
		SendJSONError(w, http.StatusConflict, "already exist")
		return
	}

	cl = h.Repo.AddClient(req.ClientID, req.Capacity, req.RPS)
	resp := clientResponse{
		ClientID:      cl.ID,
		Capacity:      cl.TokenBucket.Capacity,
		CurrentTokens: cl.TokenBucket.CurrentTokens,
		RPS:           cl.TokenBucket.RPS,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GET /clients?client_id=…
func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("client_id")
	if id == "" {
		SendJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	cl := h.Repo.GetClient(id)
	if cl == nil {
		SendJSONError(w, http.StatusNotFound, "client not found")
		return
	}
	resp := clientResponse{
		ClientID:      cl.ID,
		Capacity:      cl.TokenBucket.Capacity,
		CurrentTokens: cl.TokenBucket.CurrentTokens,
		RPS:           cl.TokenBucket.RPS,
	}
	json.NewEncoder(w).Encode(resp)
}

// PUT /clients
func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req clientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.ClientID == "" {
		SendJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	cl, err := h.Repo.UpdateClient(req.ClientID, req.Capacity, req.RPS)
	if err != nil {
		SendJSONError(w, http.StatusNotFound, "client not found")
		return
	}
	resp := clientResponse{
		ClientID:      cl.ID,
		Capacity:      cl.TokenBucket.Capacity,
		CurrentTokens: cl.TokenBucket.CurrentTokens,
		RPS:           cl.TokenBucket.RPS,
	}
	json.NewEncoder(w).Encode(resp)
}

// DELETE /clients
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("client_id")
	if id == "" {
		SendJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	if err := h.Repo.DeleteClient(id); err != nil {
		SendJSONError(w, http.StatusNotFound, "client not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
