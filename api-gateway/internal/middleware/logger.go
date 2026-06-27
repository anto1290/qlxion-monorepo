package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/qlxion/qlxion-monorepo/pkg/logger"
)

// RequestLogger middleware logs HTTP requests
func RequestLogger(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate request ID if not present
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Set request ID in response header
			w.Header().Set("X-Request-ID", requestID)

			// Create response wrapper to capture status code
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(ww, r)

			// Extract user info from context if available
			var userID, tenantID string
			if claims, ok := r.Context().Value("claims").(map[string]string); ok {
				userID = claims["user_id"]
				tenantID = claims["tenant_id"]
			}

			// Log request
			log.LogRequest(logger.RequestLog{
				Method:    r.Method,
				Path:      r.URL.Path,
				Status:    ww.statusCode,
				Duration:  time.Since(start),
				RequestID: requestID,
				UserID:    userID,
				TenantID:  tenantID,
				ClientIP:  r.RemoteAddr,
				UserAgent: r.UserAgent(),
			})
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write captures the response
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
