package httpx

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type rateLimitVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu sync.Mutex

	visitors map[string]*rateLimitVisitor

	limit rate.Limit
	burst int

	visitorTTL  time.Duration
	lastCleanup time.Time
}

func NewIPRateLimiter(
	limit rate.Limit,
	burst int,
	visitorTTL time.Duration,
) *IPRateLimiter {
	return &IPRateLimiter{
		visitors:    make(map[string]*rateLimitVisitor),
		limit:       limit,
		burst:       burst,
		visitorTTL:  visitorTTL,
		lastCleanup: time.Now(),
	}
}

func (l *IPRateLimiter) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)

			if !l.allow(ip) {
				WriteError(
					w,
					http.StatusTooManyRequests,
					"too many requests",
				)
				return
			}

			next.ServeHTTP(w, r)
		},
	)
}

func (l *IPRateLimiter) allow(ip string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastCleanup) >= l.visitorTTL {
		l.cleanup(now)
		l.lastCleanup = now
	}

	visitor, exists := l.visitors[ip]
	if !exists {
		visitor = &rateLimitVisitor{
			limiter: rate.NewLimiter(
				l.limit,
				l.burst,
			),
		}

		l.visitors[ip] = visitor
	}

	visitor.lastSeen = now

	return visitor.limiter.Allow()
}

func (l *IPRateLimiter) cleanup(now time.Time) {
	for ip, visitor := range l.visitors {
		if now.Sub(visitor.lastSeen) > l.visitorTTL {
			delete(l.visitors, ip)
		}
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(
		r.RemoteAddr,
	)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
