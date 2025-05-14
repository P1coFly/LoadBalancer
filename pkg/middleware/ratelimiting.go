package middleware

import (
	"net"
	"net/http"

	"log/slog"

	"github.com/P1coFly/LoadBalancer/pkg/client"
	"github.com/P1coFly/LoadBalancer/pkg/handlers"
)

const (
	RateLimit = "rate limit exceeded"
)

// RateLimitMiddleware проверяет и обновляет capacity у клиента
func RateLimitMiddleware(repo client.ClientRepo, logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Определяем client ID (например, из заголовка)
		clientID := extractClientIP(r)
		if clientID == "" {
			handlers.SendJSONError(w, http.StatusBadRequest, "cannot determine client IP")
			return
		}

		remaining := repo.Consume(clientID, 1)
		if !remaining {
			handlers.SendJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
			logger.Info("rate limit exceeded", "client", clientID)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// если не удалось, возвращаем весь RemoteAddr
		return r.RemoteAddr
	}
	return host
}
