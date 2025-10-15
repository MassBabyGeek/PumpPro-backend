package middleware

import (
	"net/http"
	"time"

	"github.com/MassBabyGeek/PumpPro-backend/internal/utils"
	"github.com/fatih/color"
)

// LoggerMiddleware log toutes les requêtes HTTP, même les 404
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		utils.LogRequest(r.Method, r.URL.Path, r.RemoteAddr)

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		status := wrapped.statusCode

		color.Yellow("[INFO] %s %s → %d (%v)", r.Method, r.URL.Path)

		switch {
		case status >= 500:
			color.Red("[SERVER ERROR] %s %s → %d (%v)", r.Method, r.URL.Path, status, duration)
		case status >= 400:
			color.Red("[CLIENT ERROR] %s %s → %d (%v)", r.Method, r.URL.Path, status, duration)
		default:
			color.Green("[OK] %s %s → %d (%v)", r.Method, r.URL.Path, status, duration)
		}
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
