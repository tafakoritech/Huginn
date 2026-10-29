package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tafakorpardazan/huginn/internal/passwd"
	"github.com/tafakorpardazan/huginn/internal/users"
)

// Service orchestrates authentication flows.
type Service struct {
	jwt  *Manager
	repo *users.Repository
}

func NewAuthService(jwt *Manager, repo *users.Repository) *Service {
	return &Service{jwt: jwt, repo: repo}
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
	ErrInvalidRefresh     = errors.New("invalid or expired refresh token")
)

// LoginParams carries a login attempt.
type LoginParams struct {
	Login     string // username, email or mobile
	Password  string
	Device    string
	IP        string
	UserAgent string
}

// LoginResult is returned on successful login.
type LoginResult struct {
	User         *users.User
	Tokens       Tokens
	SessionID    string
}

func (s *Service) Login(ctx context.Context, p LoginParams) (*LoginResult, error) {
	u, err := s.repo.GetByUsernameOrEmailOrMobile(ctx, p.Login)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !passwd.VerifyPassword(u.PasswordHash, p.Password) {
		return nil, ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, ErrUserInactive
	}

	tokens, sessionID, err := s.createSession(ctx, u, p)
	if err != nil {
		return nil, err
	}
	_ = s.repo.TouchLastLogin(ctx, u.ID)
	return &LoginResult{User: u, Tokens: tokens, SessionID: sessionID}, nil
}

func (s *Service) createSession(ctx context.Context, u *users.User, p LoginParams) (Tokens, string, error) {
	access, err := s.jwt.IssueAccessToken(u.ID, u.Username, u.Role)
	if err != nil {
		return Tokens{}, "", err
	}
	refresh, err := s.jwt.IssueRefreshToken()
	if err != nil {
		return Tokens{}, "", err
	}
	sess := &users.Session{
		UserID:      u.ID,
		RefreshHash: passwd.HashRefreshToken(refresh),
		Device:      strptr(p.Device),
		IP:          strptr(p.IP),
		UserAgent:   strptr(p.UserAgent),
		ExpiresAt:   time.Now().Add(s.jwt.RefreshTTL()),
	}
	created, err := s.repo.CreateSession(ctx, sess)
	if err != nil {
		return Tokens{}, "", err
	}
	return Tokens{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    s.jwt.cfg.AccessTTL,
		TokenType:    "Bearer",
	}, created.ID, nil
}

// Refresh rotates a refresh token. Returns new tokens; the old session is rotated.
func (s *Service) Refresh(ctx context.Context, refreshToken, device, ip, userAgent string) (*LoginResult, error) {
	if len(refreshToken) < 32 {
		return nil, ErrInvalidRefresh
	}
	hash := passwd.HashRefreshToken(refreshToken)
	sess, err := s.repo.GetSessionByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, users.ErrNotFound) || errors.Is(err, users.ErrSessionGone) {
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}
	if sess.RevokedAt != nil {
		return nil, ErrInvalidRefresh
	}
	if time.Now().After(sess.ExpiresAt) {
		_ = s.repo.RevokeSession(ctx, sess.ID)
		return nil, ErrInvalidRefresh
	}
	u, err := s.repo.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, ErrInvalidRefresh
	}
	if !u.IsActive {
		return nil, ErrUserInactive
	}

	// Rotate: revoke old session, create a new one.
	_ = s.repo.RevokeSession(ctx, sess.ID)
	_ = s.repo.TouchSession(ctx, sess.ID)
	tokens, newSessionID, err := s.createSession(ctx, u, LoginParams{
		Device: device, IP: ip, UserAgent: userAgent,
	})
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, Tokens: tokens, SessionID: newSessionID}, nil
}

// Logout revokes the refresh token's session.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if len(refreshToken) < 32 {
		return nil
	}
	hash := passwd.HashRefreshToken(refreshToken)
	sess, err := s.repo.GetSessionByHash(ctx, hash)
	if err != nil {
		return nil // idempotent
	}
	_ = s.repo.RevokeSession(ctx, sess.ID)
	return nil
}

// RevokeAll revokes every session for a user.
func (s *Service) RevokeAll(ctx context.Context, userID string) error {
	return s.repo.RevokeAllUserSessions(ctx, userID)
}

func (s *Service) AccessTTL() time.Duration { return s.jwt.cfg.AccessTTL }
func (s *Service) Manager() *Manager        { return s.jwt }

func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var _ = fmt.Sprintf
