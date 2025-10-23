package middleware

import (
	"net/http"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/logger"
)

// LoggerMiddleware log toutes les requêtes HTTP avec un seul log propre et coloré
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrapper pour capturer le status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Exécuter la requête
		next.ServeHTTP(wrapped, r)

		// Logger une seule fois avec toutes les infos
		duration := time.Since(start)
		logger.Request(r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wrapper pour capturer le status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
