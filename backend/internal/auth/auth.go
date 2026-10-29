package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tafakorpardazan/huginn/internal/config"
	"github.com/tafakorpardazan/huginn/internal/passwd"
	"github.com/tafakorpardazan/huginn/internal/rbac"
)

// Role aliases the shared rbac role type.
type Role = rbac.Role

// Role constants.
const (
	RoleSuperAdmin      = rbac.RoleSuperAdmin
	RoleSalesDevManager = rbac.RoleSalesDevManager
	RoleSalesAgent      = rbac.RoleSalesAgent
	RoleViewer          = rbac.RoleViewer
)

// HasPermission reports whether role permits permission.
func HasPermission(role Role, permission string) bool {
	return rbac.HasPermission(role, permission)
}

// TokenClaims is the JWT payload.
type TokenClaims struct {
	UserID   string `json:"uid"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
	jwt.RegisteredClaims
}

// Tokens is a pair of access + refresh token values.
type Tokens struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    time.Duration `json:"expires_in"`
	TokenType    string        `json:"token_type"`
}

// Manager issues and validates JWTs.
type Manager struct {
	cfg config.AuthConfig
}

func NewManager(cfg config.AuthConfig) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) IssueAccessToken(userID, username string, role Role) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.cfg.Issuer,
			Audience:  jwt.ClaimStrings{m.cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.cfg.AccessTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.cfg.AccessSecret))
}

// IssueRefreshToken returns a random opaque refresh token (not JWT) suitable
// for hashing at rest.
func (m *Manager) IssueRefreshToken() (string, error) {
	return passwd.RandomToken(48)
}

func (m *Manager) ParseAccessToken(raw string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &TokenClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(m.cfg.AccessSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// RefreshTTL returns the configured refresh token lifetime.
func (m *Manager) RefreshTTL() time.Duration { return m.cfg.RefreshTTL }
