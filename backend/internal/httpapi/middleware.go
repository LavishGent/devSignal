package httpapi

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// Middleware wraps an http.Handler with cross-cutting behavior.
type Middleware func(http.Handler) http.Handler

// Chain composes middlewares so the first listed runs outermost.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// Recover converts panics into 500 responses instead of dropped connections.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", "panic", rec, "path", r.URL.Path)
					respondError(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger emits one structured line per request.
func RequestLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// SecurityHeaders sets defense-in-depth browser protections on every
// response. HSTS is added only in production, where TLS terminates upstream.
func SecurityHeaders(production bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Content-Security-Policy",
				"default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'")
			if production {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter applies a fixed-window per-client request cap. It protects the
// credential endpoints from brute-force attempts. State is unexported; the
// only operation is wrapping a handler.
type RateLimiter struct {
	mu      sync.Mutex
	seen    map[string]*window
	limit   int
	period  time.Duration
	now     func() time.Time
	lastGC  time.Time
}

type window struct {
	start time.Time
	count int
}

// NewRateLimiter allows limit requests per period per client IP.
func NewRateLimiter(limit int, period time.Duration) *RateLimiter {
	return &RateLimiter{
		seen:   map[string]*window{},
		limit:  limit,
		period: period,
		now:    time.Now,
	}
}

// WithClock overrides the clock for tests.
func (rl *RateLimiter) WithClock(now func() time.Time) *RateLimiter {
	rl.now = now
	return rl
}

// Allow records a request for key and reports whether it is within limits.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	rl.gcLocked(now)

	w, ok := rl.seen[key]
	if !ok || now.Sub(w.start) >= rl.period {
		rl.seen[key] = &window{start: now, count: 1}
		return true
	}
	w.count++
	return w.count <= rl.limit
}

// gcLocked drops stale windows so the map cannot grow without bound.
func (rl *RateLimiter) gcLocked(now time.Time) {
	if now.Sub(rl.lastGC) < rl.period {
		return
	}
	for k, w := range rl.seen {
		if now.Sub(w.start) >= rl.period {
			delete(rl.seen, k)
		}
	}
	rl.lastGC = now
}

// Middleware rejects over-limit clients with 429.
func (rl *RateLimiter) Middleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if !rl.Allow(ip) {
				w.Header().Set("Retry-After", rl.period.String())
				respondError(w, http.StatusTooManyRequests, "too many attempts; try again later")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
