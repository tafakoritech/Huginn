package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/cache"
)

type mwKey int

const requestIDKey mwKey = iota

// RequestIDFromContext returns the request id from a request context.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// Middleware is a collection of reusable HTTP middlewares.
type Middleware struct {
	log   *slog.Logger
	cache *cache.Client
	cors  []string
}

func NewMiddleware(log *slog.Logger, cache *cache.Client, allowedOrigins []string) *Middleware {
	return &Middleware{log: log, cache: cache, cors: allowedOrigins}
}

// RequestID injects a request id into the context and response headers.
func (m *Middleware) RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = newRequestID()
		}
		w.Header().Set("X-Request-Id", rid)
		ctx := context.WithValue(r.Context(), requestIDKey, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logging emits a structured access log per request.
func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)

		attrs := []any{
			"request_id", RequestIDFromContext(r.Context()),
			"route", r.URL.Path,
			"method", r.Method,
			"status", sw.status,
			"latency_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		}
		if p, ok := auth.FromContext(r.Context()); ok {
			attrs = append(attrs, "user_id", p.UserID)
		}
		m.log.Info("http_request", attrs...)
	})
}

// Recover converts panics into 500 responses.
func (m *Middleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				m.log.Error("panic", "value", rec, "stack", string(debug.Stack()), "request_id", RequestIDFromContext(r.Context()))
				RespondError(w, http.StatusInternalServerError, &APIError{
					Code:    CodeInternalError,
					Message: "internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS handles cross-origin requests.
func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if m.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id, X-Refresh-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders sets safe defaults.
func (m *Middleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}

// RateLimit enforces a per-IP fixed-window limit.
func (m *Middleware) RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.cache == nil {
				next.ServeHTTP(w, r)
				return
			}
			ip := clientIP(r)
			ok, err := m.cache.Allow(r.Context(), "ip:"+ip, limit, window)
			if err != nil {
				m.log.Warn("rate limit check failed", "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if !ok {
				RespondError(w, http.StatusTooManyRequests, &APIError{
					Code:    CodeRateLimited,
					Message: "too many requests",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (m *Middleware) originAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	for _, o := range m.cors {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		for _, part := range splitComma(fwd) {
			if part != "" {
				return part
			}
		}
	}
	return r.RemoteAddr
}

func splitComma(s string) []string {
	var out []string
	cur := ""
	for _, c := range s {
		if c == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(c)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
