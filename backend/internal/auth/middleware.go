package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/tafakorpardazan/huginn/internal/rbac"
	"github.com/tafakorpardazan/huginn/internal/users"
)

// CtxKey is the context key for the authenticated principal.
type ctxKey int

const (
	principalKey ctxKey = iota
	requestIDKey
)

// Principal is the authenticated user placed in the request context.
type Principal struct {
	UserID   string
	Username string
	Role     rbac.Role
}

// FromContext returns the principal from a request context.
func FromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalKey).(*Principal)
	return p, ok
}

// WithPrincipal returns a copy of ctx carrying the given principal. It exists
// so tests can exercise handlers that read auth.FromContext without running a
// full login flow. Runtime code should always go through the Authenticate
// middleware instead.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// RequestID returns the request id from context.
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// Middleware authenticates Bearer tokens and enforces role permissions.
type Middleware struct {
	jwt  *Manager
	repo *users.Repository
}

func NewMiddleware(jwt *Manager, repo *users.Repository) *Middleware {
	return &Middleware{jwt: jwt, repo: repo}
}

// Authenticate validates the Authorization header and loads the principal.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeAuthError(w, "UNAUTHORIZED", "missing bearer token")
			return
		}
		raw := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.jwt.ParseAccessToken(raw)
		if err != nil {
			writeAuthError(w, "INVALID_TOKEN", "invalid or expired access token")
			return
		}
		// Always re-check user status server-side.
		u, err := m.repo.GetByID(r.Context(), claims.UserID)
		if err != nil || !u.IsActive {
			writeAuthError(w, "UNAUTHORIZED", "user not found or inactive")
			return
		}
		ctx := context.WithValue(r.Context(), principalKey, &Principal{
			UserID:   u.ID,
			Username: u.Username,
			Role:     u.Role,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission checks the principal's role against a permission.
func (m *Middleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := FromContext(r.Context())
			if !ok {
				writeAuthError(w, "UNAUTHORIZED", "authentication required")
				return
			}
			if !rbac.HasPermission(p.Role, permission) {
				writeAuthError(w, "FORBIDDEN", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole restricts to one of the given roles.
func (m *Middleware) RequireRole(roles ...Role) func(http.Handler) http.Handler {
	allowed := map[Role]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := FromContext(r.Context())
			if !ok || !allowed[p.Role] {
				writeAuthError(w, "FORBIDDEN", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

var errNoPrincipal = errors.New("no principal in context")
