package users

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/rbac"
)

// User is a Huginn account.
type User struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Username     string     `json:"username"`
	Email        *string    `json:"email"`
	Mobile       *string    `json:"mobile"`
	PasswordHash string     `json:"-"`
	Role         rbac.Role  `json:"role"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

// Session is a device/refresh-token session.
type Session struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	RefreshHash  string     `json:"-"`
	Device       *string    `json:"device"`
	IP           *string    `json:"ip"`
	UserAgent    *string    `json:"user_agent"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   time.Time  `json:"last_used_at"`
	RevokedAt    *time.Time `json:"revoked_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
}

var (
	ErrNotFound     = errors.New("user not found")
	ErrConflict     = errors.New("username/email/mobile conflict")
	ErrInactive     = errors.New("user is inactive")
	ErrSessionGone  = errors.New("session not found or revoked")
	ErrTokenMismatch = errors.New("refresh token mismatch")
)

// Repository persists users and sessions.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

const userCols = `id, name, username, email, mobile, password_hash, role, is_active, created_at, updated_at, last_login_at`

func scanUser(row database.RowScanner) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Username, &u.Email, &u.Mobile, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) Create(ctx context.Context, u *User) (*User, error) {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO users (id, name, username, email, mobile, password_hash, role, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT DO NOTHING
		RETURNING `+userCols,
		u.ID, u.Name, u.Username, u.Email, u.Mobile, u.PasswordHash, u.Role, u.IsActive)
	created, err := scanUser(row)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1 AND deleted_at IS NULL`, id)
	u, err := scanUser(row)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE username=$1 AND deleted_at IS NULL`, username)
	u, err := scanUser(row)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *Repository) GetByUsernameOrEmailOrMobile(ctx context.Context, login string) (*User, error) {
	row := r.db.Pool().QueryRow(ctx, `
		SELECT `+userCols+` FROM users
		WHERE (username=$1 OR email=$1 OR mobile=$1) AND deleted_at IS NULL`, login)
	u, err := scanUser(row)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *Repository) List(ctx context.Context, role *rbac.Role, active *bool) ([]*User, error) {
	q := `SELECT ` + userCols + ` FROM users WHERE deleted_at IS NULL`
	args := []any{}
	argc := 1
	if role != nil {
		q += ` AND role=$` + itoa(argc)
		args = append(args, *role)
		argc++
	}
	if active != nil {
		q += ` AND is_active=$` + itoa(argc)
		args = append(args, *active)
	}
	q += ` ORDER BY created_at`
	rows, err := r.db.Pool().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, u *User) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE users SET name=$2, email=$3, mobile=$4, role=$5, is_active=$6, updated_at=now()
		WHERE id=$1`, u.ID, u.Name, u.Email, u.Mobile, u.Role, u.IsActive)
	return err
}

func (r *Repository) SetPassword(ctx context.Context, userID, hash string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1`, userID, hash)
	return err
}

func (r *Repository) TouchLastLogin(ctx context.Context, userID string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE users SET last_login_at=now() WHERE id=$1`, userID)
	return err
}

func (r *Repository) Deactivate(ctx context.Context, userID string) error {
	res, err := r.db.Pool().Exec(ctx, `UPDATE users SET is_active=false, updated_at=now() WHERE id=$1`, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	// revoke all sessions
	_, err = r.db.Pool().Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return err
}

func (r *Repository) Reactivate(ctx context.Context, userID string) error {
	res, err := r.db.Pool().Exec(ctx, `UPDATE users SET is_active=true, updated_at=now() WHERE id=$1`, userID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- Sessions ----

func (r *Repository) CreateSession(ctx context.Context, s *Session) (*Session, error) {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO sessions (id, user_id, refresh_hash, device, ip, user_agent, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, user_id, refresh_hash, device, ip, user_agent, created_at, last_used_at, revoked_at, expires_at`,
		s.ID, s.UserID, s.RefreshHash, s.Device, s.IP, s.UserAgent, s.ExpiresAt)
	return scanSession(row)
}

func (r *Repository) GetSessionByHash(ctx context.Context, refreshHash string) (*Session, error) {
	row := r.db.Pool().QueryRow(ctx, `
		SELECT id, user_id, refresh_hash, device, ip, user_agent, created_at, last_used_at, revoked_at, expires_at
		FROM sessions WHERE refresh_hash=$1`, refreshHash)
	return scanSession(row)
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE id=$1 AND revoked_at IS NULL`, sessionID)
	return err
}

func (r *Repository) GetSessionByID(ctx context.Context, sessionID string) (*Session, error) {
	row := r.db.Pool().QueryRow(ctx, `
		SELECT id, user_id, refresh_hash, device, ip, user_agent, created_at, last_used_at, revoked_at, expires_at
		FROM sessions WHERE id=$1`, sessionID)
	return scanSession(row)
}

func (r *Repository) RevokeAllUserSessions(ctx context.Context, userID string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return err
}

func (r *Repository) TouchSession(ctx context.Context, sessionID string) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE sessions SET last_used_at=now() WHERE id=$1`, sessionID)
	return err
}

func (r *Repository) ListSessions(ctx context.Context, userID string) ([]*Session, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, user_id, refresh_hash, device, ip, user_agent, created_at, last_used_at, revoked_at, expires_at
		FROM sessions WHERE user_id=$1 ORDER BY last_used_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Session{}
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanSession(row database.RowScanner) (*Session, error) {
	var s Session
	err := row.Scan(&s.ID, &s.UserID, &s.RefreshHash, &s.Device, &s.IP, &s.UserAgent, &s.CreatedAt, &s.LastUsedAt, &s.RevokedAt, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
