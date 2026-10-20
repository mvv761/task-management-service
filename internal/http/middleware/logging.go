package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const requestIDHeader = "X-Request-ID"

var fallbackRequestIDCounter atomic.Uint64

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.status = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(body)
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if requestID == "" {
			requestID = newRequestID()
		}

		w.Header().Set(requestIDHeader, requestID)

		meta := &requestMetadata{
			requestID: requestID,
		}
		ctx := contextWithRequestID(r.Context(), requestID, meta)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			attrs := []any{
				"request_id", requestIDForLog(r),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", time.Since(start).Milliseconds(),
			}

			if meta, ok := requestMetadataFromContext(r.Context()); ok && meta.hasUserID {
				attrs = append(attrs, "user_id", meta.userID)
			}

			logger.Info("request completed", attrs...)
		})
	}
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}

	return fmt.Sprintf("fallback-%d-%d", time.Now().UnixNano(), fallbackRequestIDCounter.Add(1))
}

func requestIDForLog(r *http.Request) string {
	if requestID, ok := RequestIDFromContext(r.Context()); ok {
		return requestID
	}

	return ""
}
