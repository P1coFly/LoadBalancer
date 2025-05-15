package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/P1coFly/LoadBalancer/pkg/client"
)

const (
	ErrNoClient = "client not found"
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
		h.Logger.Error("Create client - can't decode body", "err", err)
		SendJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.ClientID == "" {
		h.Logger.Error("Create client - no client id in body")
		SendJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	cl := h.Repo.GetClient(req.ClientID)
	if cl != nil {
		h.Logger.Error("Create client - already exist")
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.Error("Create client - fail to send clientResponse", "err", err)
		SendJSONError(w, http.StatusInternalServerError, "fail to send response")
	}
}

// GET /clients?client_id=…
func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("client_id")
	if id == "" {
		h.Logger.Error("Get client - no client id in body")
		SendJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	cl := h.Repo.GetClient(id)
	if cl == nil {
		h.Logger.Error("Get client", "err", ErrNoClient)
		SendJSONError(w, http.StatusNotFound, ErrNoClient)
		return
	}
	resp := clientResponse{
		ClientID:      cl.ID,
		Capacity:      cl.TokenBucket.Capacity,
		CurrentTokens: cl.TokenBucket.CurrentTokens,
		RPS:           cl.TokenBucket.RPS,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.Error("Get client - fail to send clientResponse", "err", err)
		SendJSONError(w, http.StatusInternalServerError, "fail to send response")
	}
}

// PUT /clients
func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req clientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("Update client - can't decode body", "err", err)
		SendJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.ClientID == "" {
		h.Logger.Error("Update client - no client id in body")
		SendJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	cl, err := h.Repo.UpdateClient(req.ClientID, req.Capacity, req.RPS)
	if err != nil {
		h.Logger.Error("Update client", "err", ErrNoClient)
		SendJSONError(w, http.StatusNotFound, ErrNoClient)
		return
	}
	resp := clientResponse{
		ClientID:      cl.ID,
		Capacity:      cl.TokenBucket.Capacity,
		CurrentTokens: cl.TokenBucket.CurrentTokens,
		RPS:           cl.TokenBucket.RPS,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.Logger.Error("Update client - fail to send clientResponse", "err", err)
		SendJSONError(w, http.StatusInternalServerError, "fail to send response")
	}
}

// DELETE /clients?client_id=…
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("client_id")
	if id == "" {
		h.Logger.Error("Get client - no client id in body")
		SendJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	if err := h.Repo.DeleteClient(id); err != nil {
		h.Logger.Error("Delete client", "err", ErrNoClient)
		SendJSONError(w, http.StatusNotFound, ErrNoClient)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
