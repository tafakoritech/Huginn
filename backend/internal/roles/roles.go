// Package roles manages the role definitions and their permissions. The four
// built-in roles are seeded from the real RBAC matrices in internal/rbac so the
// API (and the dashboard) always reflects the actual authorization model.
package roles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/rbac"
)

// Role is a persisted role definition.
type Role struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsSystem    bool      `json:"is_system"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var (
	ErrNotFound    = errors.New("role not found")
	ErrExists      = errors.New("role already exists")
	ErrSystemRole  = errors.New("system role cannot be modified in that way")
	ErrInUse       = errors.New("role is assigned to users and cannot be deleted")
	ErrInvalidKey  = errors.New("role key must match ^[a-z][a-z0-9_]*$")
)

func validKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9' && i > 0) {
			continue
		}
		return false
	}
	return true
}

// Repository persists role definitions.
type Repository struct {
	db      *database.DB
	ensure  sync.Once
	ensureErr error
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// Ensure creates the roles table and seeds the built-in roles from the real
// RBAC matrices. Safe to call repeatedly; seeds are idempotent.
func (r *Repository) Ensure(ctx context.Context) error {
	r.ensure.Do(func() {
		r.ensureErr = r.ensureOnce(ctx)
	})
	return r.ensureErr
}

func (r *Repository) ensureOnce(ctx context.Context) error {
	// Allow users to reference custom roles (system enum -> TEXT). No-op once TEXT.
	if _, err := r.db.Pool().Exec(ctx, `ALTER TABLE users ALTER COLUMN role TYPE TEXT`); err != nil {
		return fmt.Errorf("alter users.role: %w", err)
	}
	if _, err := r.db.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS roles (
			key         TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			description TEXT,
			is_system   BOOLEAN NOT NULL DEFAULT false,
			permissions JSONB NOT NULL DEFAULT '[]',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create roles table: %w", err)
	}
	names := map[rbac.Role]string{
		rbac.RoleSuperAdmin:      "Super Admin",
		rbac.RoleSalesDevManager: "Sales Development Manager",
		rbac.RoleSalesAgent:      "Sales Agent",
		rbac.RoleViewer:          "Viewer",
	}
	for role, perms := range rbac.StaticRoleDefinitions() {
		pb, _ := json.Marshal(perms)
		if _, err := r.db.Pool().Exec(ctx, `
			INSERT INTO roles (key, name, description, is_system, permissions)
			VALUES ($1,$2,$3,true,$4)
			ON CONFLICT (key) DO UPDATE SET name=EXCLUDED.name, is_system=true, permissions=EXCLUDED.permissions, updated_at=now()`,
			string(role), names[role], nil, pb); err != nil {
			return fmt.Errorf("seed role %s: %w", role, err)
		}
	}
	return nil
}

const cols = `key, name, description, is_system, permissions, created_at, updated_at`

func scan(row database.RowScanner) (*Role, error) {
	var r Role
	var perms []byte
	if err := row.Scan(&r.Key, &r.Name, &r.Description, &r.IsSystem, &perms, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(perms, &r.Permissions)
	if r.Permissions == nil {
		r.Permissions = []string{}
	}
	return &r, nil
}

func (r *Repository) List(ctx context.Context) ([]*Role, error) {
	if err := r.Ensure(ctx); err != nil {
		return nil, err
	}
	rows, err := r.db.Pool().Query(ctx, `SELECT `+cols+` FROM roles ORDER BY is_system DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Role{}
	for rows.Next() {
		role, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, key string) (*Role, error) {
	if err := r.Ensure(ctx); err != nil {
		return nil, err
	}
	row := r.db.Pool().QueryRow(ctx, `SELECT `+cols+` FROM roles WHERE key=$1`, key)
	role, err := scan(row)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return role, nil
}

func (r *Repository) Create(ctx context.Context, role *Role) error {
	if err := r.Ensure(ctx); err != nil {
		return err
	}
	pb, _ := json.Marshal(role.Permissions)
	_, err := r.db.Pool().Exec(ctx, `
		INSERT INTO roles (key, name, description, is_system, permissions)
		VALUES ($1,$2,$3,false,$4)`,
		role.Key, role.Name, role.Description, pb)
	return err
}

func (r *Repository) Update(ctx context.Context, key string, name *string, description *string, permissions []string) error {
	if err := r.Ensure(ctx); err != nil {
		return err
	}
	colsSet := []string{"updated_at=now()"}
	args := []any{}
	argc := 1
	if name != nil {
		colsSet = append(colsSet, fmt.Sprintf("name=$%d", argc))
		args = append(args, *name)
		argc++
	}
	if description != nil {
		colsSet = append(colsSet, fmt.Sprintf("description=$%d", argc))
		args = append(args, *description)
		argc++
	}
	if permissions != nil {
		pb, _ := json.Marshal(permissions)
		colsSet = append(colsSet, fmt.Sprintf("permissions=$%d", argc))
		args = append(args, pb)
		argc++
	}
	args = append(args, key)
	res, err := r.db.Pool().Exec(ctx,
		fmt.Sprintf("UPDATE roles SET %s WHERE key=$%d", strings.Join(colsSet, ", "), argc), args...)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, key string) error {
	if err := r.Ensure(ctx); err != nil {
		return err
	}
	res, err := r.db.Pool().Exec(ctx, `DELETE FROM roles WHERE key=$1 AND is_system=false`, key)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) IsSystem(ctx context.Context, key string) (bool, error) {
	role, err := r.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return role.IsSystem, nil
}

func (r *Repository) UsersUsing(ctx context.Context, key string) (int, error) {
	if err := r.Ensure(ctx); err != nil {
		return 0, err
	}
	var n int
	err := r.db.Pool().QueryRow(ctx, `SELECT count(*) FROM users WHERE role=$1 AND deleted_at IS NULL`, key).Scan(&n)
	return n, err
}

// HasPermission reports whether the role grants a permission. Built-in roles
// always resolve against the real RBAC matrices; custom roles use the stored
// permission list.
func (r *Repository) HasPermission(ctx context.Context, roleKey, permission string) (bool, error) {
	if rbac.IsRole(rbac.Role(roleKey)) {
		return rbac.HasPermission(rbac.Role(roleKey), permission), nil
	}
	role, err := r.Get(ctx, roleKey)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	for _, p := range role.Permissions {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}

// Service exposes role CRUD with validation.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context) ([]*Role, error) { return s.repo.List(ctx) }
func (s *Service) Get(ctx context.Context, key string) (*Role, error) {
	return s.repo.Get(ctx, key)
}

type CreateParams struct {
	Key         string
	Name        string
	Description *string
	Permissions []string
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*Role, error) {
	if !validKey(p.Key) {
		return nil, ErrInvalidKey
	}
	if p.Name == "" {
		return nil, errors.New("name is required")
	}
	role := &Role{Key: p.Key, Name: p.Name, Description: p.Description, Permissions: dedupe(p.Permissions)}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, p.Key)
}

type UpdateParams struct {
	Name        *string
	Description *string
	Permissions []string
}

func (s *Service) Update(ctx context.Context, key string, p UpdateParams) (*Role, error) {
	system, err := s.repo.IsSystem(ctx, key)
	if err != nil {
		return nil, err
	}
	if system && p.Permissions != nil {
		// Built-in roles keep the permissions defined in code; only display
		// metadata (name/description) may be edited.
		p.Permissions = nil
	}
	if p.Name != nil && *p.Name == "" {
		return nil, errors.New("name cannot be empty")
	}
	perms := p.Permissions
	if perms != nil {
		perms = dedupe(perms)
	}
	if err := s.repo.Update(ctx, key, p.Name, p.Description, perms); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, key)
}

func (s *Service) Delete(ctx context.Context, key string) error {
	system, err := s.repo.IsSystem(ctx, key)
	if err != nil {
		return err
	}
	if system {
		return ErrSystemRole
	}
	n, err := s.repo.UsersUsing(ctx, key)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrInUse
	}
	return s.repo.Delete(ctx, key)
}

// HasPermission delegates to the repository.
func (s *Service) HasPermission(ctx context.Context, roleKey, permission string) (bool, error) {
	return s.repo.HasPermission(ctx, roleKey, permission)
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
