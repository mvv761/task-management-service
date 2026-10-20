package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiterRequestsWithinLimitAreAllowed(t *testing.T) {
	rl := newTestRateLimiter(t, 1, 3)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, newRateLimitRequest("10.0.0.1:1234", 0))

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	}
}

func TestRateLimiterRequestsAboveLimitReturn429(t *testing.T) {
	rl := newTestRateLimiter(t, 1, 1)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, newRateLimitRequest("10.0.0.1:1234", 0))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, newRateLimitRequest("10.0.0.1:5678", 0))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiterDifferentClientsHaveIndependentLimits(t *testing.T) {
	rl := newTestRateLimiter(t, 1, 1)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	clientAFirst := httptest.NewRecorder()
	handler.ServeHTTP(clientAFirst, newRateLimitRequest("10.0.0.1:1234", 0))
	if clientAFirst.Code != http.StatusOK {
		t.Fatalf("client A first status = %d, want %d", clientAFirst.Code, http.StatusOK)
	}

	clientASecond := httptest.NewRecorder()
	handler.ServeHTTP(clientASecond, newRateLimitRequest("10.0.0.1:5678", 0))
	if clientASecond.Code != http.StatusTooManyRequests {
		t.Fatalf("client A second status = %d, want %d", clientASecond.Code, http.StatusTooManyRequests)
	}

	clientB := httptest.NewRecorder()
	handler.ServeHTTP(clientB, newRateLimitRequest("10.0.0.2:1234", 0))
	if clientB.Code != http.StatusOK {
		t.Fatalf("client B status = %d, want %d", clientB.Code, http.StatusOK)
	}
}

func TestRateLimiterAuthenticatedUsersAreSeparatedByUserID(t *testing.T) {
	rl := newTestRateLimiter(t, 1, 1)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	userOneFirst := httptest.NewRecorder()
	handler.ServeHTTP(userOneFirst, newRateLimitRequest("10.0.0.1:1234", 1))
	if userOneFirst.Code != http.StatusOK {
		t.Fatalf("user 1 first status = %d, want %d", userOneFirst.Code, http.StatusOK)
	}

	userOneSecond := httptest.NewRecorder()
	handler.ServeHTTP(userOneSecond, newRateLimitRequest("10.0.0.2:1234", 1))
	if userOneSecond.Code != http.StatusTooManyRequests {
		t.Fatalf("user 1 second status = %d, want %d", userOneSecond.Code, http.StatusTooManyRequests)
	}

	userTwo := httptest.NewRecorder()
	handler.ServeHTTP(userTwo, newRateLimitRequest("10.0.0.1:1234", 2))
	if userTwo.Code != http.StatusOK {
		t.Fatalf("user 2 status = %d, want %d", userTwo.Code, http.StatusOK)
	}
}

func TestRateLimiterRetryAfterPresent(t *testing.T) {
	rl := newTestRateLimiter(t, 1, 1)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), newRateLimitRequest("10.0.0.1:1234", 0))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, newRateLimitRequest("10.0.0.1:1234", 0))

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTooManyRequests)
	}

	if retryAfter := rr.Header().Get("Retry-After"); retryAfter != "1" {
		t.Fatalf("Retry-After = %q, want 1", retryAfter)
	}
}

func TestRateLimiterConcurrentAccess(t *testing.T) {
	rl := newTestRateLimiter(t, 1000, 1000)

	var handled atomic.Int64
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handled.Add(1)
		w.WriteHeader(http.StatusOK)
	}))

	const requests = 100

	var wg sync.WaitGroup
	wg.Add(requests)

	for i := 0; i < requests; i++ {
		go func() {
			defer wg.Done()

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, newRateLimitRequest("10.0.0.1:1234", 0))

			if rr.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
			}
		}()
	}

	wg.Wait()

	if got := handled.Load(); got != requests {
		t.Fatalf("handled requests = %d, want %d", got, requests)
	}
}

func newTestRateLimiter(t *testing.T, rps float64, burst int) *RateLimiter {
	t.Helper()

	rl, err := newRateLimiter(rps, burst, time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new rate limiter: %v", err)
	}

	t.Cleanup(rl.Close)

	return rl
}

func newRateLimitRequest(remoteAddr string, userID uint64) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr

	if userID != 0 {
		req = req.WithContext(contextWithUserID(req.Context(), userID))
	}

	return req
}
