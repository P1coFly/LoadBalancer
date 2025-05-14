package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// AccessLog — middleware, логирующий входящие запросы.
func AccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusResponseWriter{ResponseWriter: w, status: 0}
		next.ServeHTTP(sw, r)
		logger.Info("New request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("client_ip", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.Int("status code", sw.status),
			slog.String("duration", time.Since(start).String()),
		)
	})
}

// statusResponseWriter - используется для перехвата кода ответа
type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

// Переопределяем WriteHeader — здесь и сохраняем код
func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
