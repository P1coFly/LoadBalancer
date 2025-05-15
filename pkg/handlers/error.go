package handlers

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SendJSONError — отправляет JSON-ошибку клиенту
func SendJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: message,
	})
	if err != nil {
		SendJSONError(w, http.StatusInternalServerError, "client not found")
	}
}
