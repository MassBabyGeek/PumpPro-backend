package middleware

import (
	"net/http"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
)

// LoggerMiddleware log toutes les requêtes HTTP
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log la requête entrante
		utils.LogRequest(r.Method, r.URL.Path, r.RemoteAddr)

		// Wrapper pour capturer le status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Appeler le handler suivant
		next.ServeHTTP(wrapped, r)

		// Log la durée et le status code
		duration := time.Since(start)
		utils.LogInfo("%s %s - Status: %d - Duration: %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
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
