package share

import (
	"net"
	"net/http"
	"strconv"
	"sync"

	"golang.org/x/time/rate"
)

// TooManyRequestsBody is the fixed 429 payload for share routes (Story 3.5 NFR-06).
// It must not vary with token validity (no existence oracle).
var TooManyRequestsBody = []byte("Too Many Requests\n")

type rateLimitConfig struct {
	// r is the steady-state refill rate in tokens per second (see rate.Limit).
	r     rate.Limit
	burst int
}

// defaultShareRateLimit is tuned for loopback desktop MVP: high enough that normal
// browsing and table-driven HTTP tests do not false-429, while still bounding
// unbounded abuse from a single IP. Stricter limits belong at a reverse proxy
// when the service is exposed beyond loopback (docs/share-abuse-posture.md).
func defaultShareRateLimit() rateLimitConfig {
	return rateLimitConfig{
		r:     rate.Limit(12), // 12 req/s refill
		burst: 80,
	}
}

type visitor struct {
	lim *rate.Limiter
}

// maxRateLimitVisitorEntries bounds in-memory map growth (Story 3.5 session 2).
// Distinct IP keys beyond this cap evict one arbitrary existing entry before insert.
// Loopback MVP sees one key; public/LAN exposure should still rely on edge limits.
const maxRateLimitVisitorEntries = 4096

type ipRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
}

func newIPRateLimiter() *ipRateLimiter {
	return &ipRateLimiter{visitors: make(map[string]*visitor)}
}

func (ip *ipRateLimiter) allow(hostKey string, cfg rateLimitConfig) bool {
	ip.mu.Lock()
	defer ip.mu.Unlock()
	v, ok := ip.visitors[hostKey]
	if !ok {
		if len(ip.visitors) >= maxRateLimitVisitorEntries {
			for k := range ip.visitors {
				delete(ip.visitors, k)
				break
			}
		}
		v = &visitor{lim: rate.NewLimiter(cfg.r, cfg.burst)}
		ip.visitors[hostKey] = v
	}
	return v.lim.Allow()
}

// clientIPKey uses RemoteAddr host only. X-Forwarded-For is intentionally ignored
// unless a future trusted-proxy mode exists (spoofing hazard).
func clientIPKey(r *http.Request) string {
	if r == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type rateLimitedHandler struct {
	inner http.Handler
	cfg   rateLimitConfig
	ips   *ipRateLimiter
}

func wrapRateLimitedHandler(inner http.Handler, cfg rateLimitConfig) http.Handler {
	return &rateLimitedHandler{
		inner: inner,
		cfg:   cfg,
		ips:   newIPRateLimiter(),
	}
}

func (h *rateLimitedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.ips.allow(clientIPKey(r), h.cfg) {
		writeTooManyRequests(w, r)
		return
	}
	h.inner.ServeHTTP(w, r)
}

func writeTooManyRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", strconv.Itoa(len(TooManyRequestsBody)))
	if r != nil && r.Method == http.MethodHead {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write(TooManyRequestsBody)
}
