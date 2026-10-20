package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		wantStatus    int
		requestID     string
		wantRequestID string
		wantBody      string
		wantLogUserID bool
		wantLogStatus float64
	}{
		{
			name: "implicit status ok",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("ok"))
			},
			wantStatus:    http.StatusOK,
			wantBody:      "ok",
			wantLogStatus: float64(http.StatusOK),
		},
		{
			name: "explicit status created",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			},
			wantStatus:    http.StatusCreated,
			requestID:     "req-123",
			wantRequestID: "req-123",
			wantLogStatus: float64(http.StatusCreated),
			wantLogUserID: true,
		},
		{
			name: "explicit status too many requests",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "slow down", http.StatusTooManyRequests)
			},
			wantStatus:    http.StatusTooManyRequests,
			wantBody:      "slow down\n",
			wantLogStatus: float64(http.StatusTooManyRequests),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))

			handler := http.Handler(tt.handler)
			if tt.wantLogUserID {
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					r = r.WithContext(contextWithUserID(r.Context(), 42))
					tt.handler(w, r)
				})
			}

			handler = RequestID(RequestLogging(logger)(handler))

			req := httptest.NewRequest(http.MethodPost, "/tasks?token=redacted", nil)
			if tt.requestID != "" {
				req.Header.Set(requestIDHeader, tt.requestID)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			if tt.wantBody != "" && rr.Body.String() != tt.wantBody {
				t.Fatalf("body = %q, want %q", rr.Body.String(), tt.wantBody)
			}

			responseRequestID := rr.Header().Get(requestIDHeader)
			if responseRequestID == "" {
				t.Fatal("response request id is empty")
			}

			if tt.wantRequestID != "" && responseRequestID != tt.wantRequestID {
				t.Fatalf("response request id = %q, want %q", responseRequestID, tt.wantRequestID)
			}

			logRecord := readLogRecord(t, &logs)

			if logRecord["msg"] != "request completed" {
				t.Fatalf("msg = %v, want request completed", logRecord["msg"])
			}

			if logRecord["request_id"] != responseRequestID {
				t.Fatalf("request_id = %v, want %q", logRecord["request_id"], responseRequestID)
			}

			if logRecord["method"] != http.MethodPost {
				t.Fatalf("method = %v, want %s", logRecord["method"], http.MethodPost)
			}

			if logRecord["path"] != "/tasks" {
				t.Fatalf("path = %v, want /tasks", logRecord["path"])
			}

			if _, exists := logRecord["query"]; exists {
				t.Fatal("request log must not contain query string")
			}

			if logRecord["status"] != tt.wantLogStatus {
				t.Fatalf("status = %v, want %v", logRecord["status"], tt.wantLogStatus)
			}

			if _, exists := logRecord["duration_ms"]; !exists {
				t.Fatal("duration_ms is missing")
			}

			if tt.wantLogUserID && logRecord["user_id"] != float64(42) {
				t.Fatalf("user_id = %v, want 42", logRecord["user_id"])
			}
		})
	}
}

func readLogRecord(t *testing.T, logs *bytes.Buffer) map[string]any {
	t.Helper()

	scanner := bufio.NewScanner(logs)
	if !scanner.Scan() {
		t.Fatal("missing log record")
	}

	var record map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal log record: %v", err)
	}

	return record
}
