package coordinator

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		return nil
	}
	return &rateLimiter{
		buckets: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
}

func (r *rateLimiter) allow(key string) bool {
	if r == nil {
		return true
	}
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	cutoff := now.Add(-r.window)
	times := r.buckets[key]
	i := 0
	for _, t := range times {
		if t.After(cutoff) {
			times[i] = t
			i++
		}
	}
	times = times[:i]
	if len(times) >= r.limit {
		r.buckets[key] = times
		return false
	}
	times = append(times, now)
	r.buckets[key] = times
	return true
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func ipAllowed(ip string, cidrs []string) bool {
	if len(cidrs) == 0 {
		return false
	}
	addr := net.ParseIP(ip)
	if addr == nil {
		return false
	}
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(strings.TrimSpace(c))
		if err != nil {
			continue
		}
		if n.Contains(addr) {
			return true
		}
	}
	return false
}

func (s *Server) rateLimit(lim *rateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if ipAllowed(ip, s.cfg.AllowCIDR) {
			next(w, r)
			return
		}
		if !lim.allow(ip) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}