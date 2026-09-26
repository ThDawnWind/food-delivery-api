package httpx

import (
	"net/http"
	"net/netip"
	"strings"
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

	trustedProxyCIDRs []netip.Prefix
}

func NewIPRateLimiter(
	limit rate.Limit,
	burst int,
	visitorTTL time.Duration,
	trustedProxyCIDRs []netip.Prefix,
) *IPRateLimiter {
	return &IPRateLimiter{
		visitors: make(
			map[string]*rateLimitVisitor,
		),
		limit:       limit,
		burst:       burst,
		visitorTTL:  visitorTTL,
		lastCleanup: time.Now(),
		trustedProxyCIDRs: append(
			[]netip.Prefix(nil),
			trustedProxyCIDRs...,
		),
	}
}

func (l *IPRateLimiter) Middleware(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(
				r,
				l.trustedProxyCIDRs,
			)

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

func clientIP(
	r *http.Request,
	trustedProxyCIDRs []netip.Prefix,
) string {
	remoteIP, ok := parseRemoteIP(
		r.RemoteAddr,
	)
	if !ok {
		return r.RemoteAddr
	}

	if !isTrustedProxy(
		remoteIP,
		trustedProxyCIDRs,
	) {
		return remoteIP.String()
	}

	forwardedFor := strings.TrimSpace(
		r.Header.Get("X-Forwarded-For"),
	)

	if forwardedFor == "" {
		return remoteIP.String()
	}

	parts := strings.Split(
		forwardedFor,
		",",
	)

	for i := len(parts) - 1; i >= 0; i-- {
		rawIP := strings.TrimSpace(
			parts[i],
		)

		ip, err := netip.ParseAddr(rawIP)
		if err != nil {
			// Заголовок содержит мусор.
			// Безопаснее вообще ему не доверять.
			return remoteIP.String()
		}

		ip = ip.Unmap()

		if isTrustedProxy(
			ip,
			trustedProxyCIDRs,
		) {
			continue
		}

		return ip.String()
	}

	return remoteIP.String()
}

func parseRemoteIP(remoteAddr string) (netip.Addr, bool) {
	addrPort, err := netip.ParseAddrPort(
		remoteAddr,
	)
	if err == nil {
		return addrPort.Addr().Unmap(), true
	}

	addr, err := netip.ParseAddr(
		remoteAddr,
	)
	if err != nil {
		return netip.Addr{}, false
	}

	return addr.Unmap(), true
}

func isTrustedProxy(
	ip netip.Addr,
	trustedProxyCIDRs []netip.Prefix,
) bool {
	for _, prefix := range trustedProxyCIDRs {
		if prefix.Contains(ip) {
			return true
		}
	}

	return false
}
