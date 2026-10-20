package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultRateLimitStaleAfter   = 10 * time.Minute
	defaultRateLimitCleanupEvery = time.Minute
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	limit        rate.Limit
	burst        int
	staleAfter   time.Duration
	cleanupEvery time.Duration

	mu      sync.Mutex
	clients map[string]*clientLimiter

	closeOnce sync.Once
	stop      chan struct{}
	done      chan struct{}
}

func NewRateLimiter(rps float64, burst int) (*RateLimiter, error) {
	return newRateLimiter(
		rps,
		burst,
		defaultRateLimitStaleAfter,
		defaultRateLimitCleanupEvery,
	)
}

func newRateLimiter(
	rps float64,
	burst int,
	staleAfter time.Duration,
	cleanupEvery time.Duration,
) (*RateLimiter, error) {
	if rps <= 0 {
		return nil, fmt.Errorf("rate limit rps must be greater than 0")
	}

	if burst <= 0 {
		return nil, fmt.Errorf("rate limit burst must be greater than 0")
	}

	if staleAfter <= 0 {
		return nil, fmt.Errorf("rate limit stale duration must be greater than 0")
	}

	if cleanupEvery <= 0 {
		return nil, fmt.Errorf("rate limit cleanup interval must be greater than 0")
	}

	rl := &RateLimiter{
		limit:        rate.Limit(rps),
		burst:        burst,
		staleAfter:   staleAfter,
		cleanupEvery: cleanupEvery,
		clients:      make(map[string]*clientLimiter),
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}

	go rl.cleanupLoop()

	return rl, nil
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := rl.limiterFor(r)
		if !limiter.Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) Close() {
	rl.closeOnce.Do(func() {
		close(rl.stop)
		<-rl.done
	})
}

func (rl *RateLimiter) limiterFor(r *http.Request) *rate.Limiter {
	key := clientKey(r)
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, ok := rl.clients[key]
	if !ok {
		client = &clientLimiter{
			limiter: rate.NewLimiter(rl.limit, rl.burst),
		}
		rl.clients[key] = client
	}

	client.lastSeen = now

	return client.limiter
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupEvery)
	defer func() {
		ticker.Stop()
		close(rl.done)
	}()

	for {
		select {
		case <-ticker.C:
			rl.cleanupStaleClients(time.Now())
		case <-rl.stop:
			return
		}
	}
}

func (rl *RateLimiter) cleanupStaleClients(now time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for key, client := range rl.clients {
		if now.Sub(client.lastSeen) > rl.staleAfter {
			delete(rl.clients, key)
		}
	}
}

func clientKey(r *http.Request) string {
	if userID, ok := UserIDFromContext(r.Context()); ok {
		return fmt.Sprintf("user:%d", userID)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	host = strings.TrimSpace(host)
	if host == "" {
		host = "unknown"
	}

	return "ip:" + host
}
