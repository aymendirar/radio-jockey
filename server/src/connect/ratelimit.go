package connect

import (
	"net"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/time/rate"
)

type ipLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiterEntry
	rps      rate.Limit
	burst    int
}

func newIPRateLimiter(rps float64, burst int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*ipLimiterEntry),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (r *ipRateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.limiters[key]
	if !ok {
		e = &ipLimiterEntry{limiter: rate.NewLimiter(r.rps, r.burst)}
		r.limiters[key] = e
	}
	e.lastSeen = time.Now()
	return e.limiter.Allow()
}

// startCleanup periodically purges limiter entries idle longer than maxIdle,
// keeping the per-IP map from growing unbounded as distinct clients churn.
func (r *ipRateLimiter) startCleanup(interval, maxIdle time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			r.mu.Lock()
			now := time.Now()
			for key, e := range r.limiters {
				if now.Sub(e.lastSeen) > maxIdle {
					delete(r.limiters, key)
				}
			}
			r.mu.Unlock()
		}
	}()
}

// clientKey identifies the calling client for rate-limiting purposes. Caddy
// sits in front of this server and appends the real peer address to
// X-Forwarded-For rather than replacing it, so the last entry is the one
// Caddy itself added and is the only one that can't be spoofed by the
// client; earlier entries may be attacker-supplied. Falls back to the direct
// TCP peer address when the header is absent (e.g. tests hitting the server
// directly, or no reverse proxy in front).
func clientKey(req connect.AnyRequest) string {
	if xff := req.Header().Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(req.Peer().Addr)
	if err != nil {
		return req.Peer().Addr
	}
	return host
}
