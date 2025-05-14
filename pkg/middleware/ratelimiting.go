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

		// Получаем или создаём клиента
		cl := repo.GetClient(clientID)
		if cl == nil {
			// Добавляем с дефолтными параметрами
			cl = repo.AddClient(clientID, repo.DefaultCapacity(), repo.DefaultRPS())
		}

		// Проверяем остаток capacity
		if cl.Capacity <= 0 {
			handlers.SendJSONError(w, http.StatusTooManyRequests, RateLimit)
			logger.Info(RateLimit, "clientID", clientID)
			return
		}

		// Уменьшаем capacity и обновляем в репо
		_, err := repo.UpdateClient(clientID, cl.Capacity-1, cl.RPS)
		if err != nil {
			// На всякий случай
			handlers.SendJSONError(w, http.StatusInternalServerError, "cannot update client capacity")
			logger.Error("cannot update client capacity ", "error", err)
			return
		}
		logger.Debug("Consumed capacity", "id", clientID, "remaining", cl.Capacity-1)
		next.ServeHTTP(w, r)
	})
}

func extractClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// если не удалось, возвращаем всё RemoteAddr
		return r.RemoteAddr
	}
	return host
}
