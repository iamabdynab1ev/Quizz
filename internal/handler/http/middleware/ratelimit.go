package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*ipLimiter
	rps     rate.Limit
	burst   int
}

func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	rl := &IPRateLimiter{
		clients: make(map[string]*ipLimiter),
		rps:     rate.Limit(rps),
		burst:   burst,
	}
	go rl.cleanup()
	return rl
}

func (rl *IPRateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, ok := rl.clients[ip]
	if !ok {
		entry = &ipLimiter{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.clients[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (rl *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		threshold := time.Now().Add(-10 * time.Minute)
		rl.mu.Lock()
		for ip, entry := range rl.clients {
			if entry.lastSeen.Before(threshold) {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func RateLimit(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if !limiter.get(ip).Allow() {
				http.Error(w, `{"error":"too_many_requests","message":"Слишком много запросов"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
