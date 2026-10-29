package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/passwd"
	"github.com/tafakorpardazan/huginn/internal/rbac"
)

// Service provides user management operations.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type CreateParams struct {
	Name     string
	Username string
	Email    *string
	Mobile   *string
	Password string
	Role     rbac.Role
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*User, error) {
	if p.Name == "" || p.Username == "" {
		return nil, errors.New("name and username are required")
	}
	if p.Password == "" {
		return nil, errors.New("password is required")
	}
	if err := validateRole(p.Role); err != nil {
		return nil, err
	}
	hash, err := passwd.HashPassword(p.Password)
	if err != nil {
		return nil, err
	}
	u := &User{
		ID:       uuid.NewString(),
		Name:     p.Name,
		Username: p.Username,
		Email:    p.Email,
		Mobile:   p.Mobile,
		Role:     p.Role,
		IsActive: true,
	}
	u.PasswordHash = hash
	return s.repo.Create(ctx, u)
}

type UpdateParams struct {
	Name   string
	Email  *string
	Mobile *string
	Role   rbac.Role
}

func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (*User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateRole(p.Role); err != nil {
		return nil, err
	}
	u.Name = p.Name
	u.Email = p.Email
	u.Mobile = p.Mobile
	u.Role = p.Role
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) ResetPassword(ctx context.Context, id, newPassword string) error {
	if newPassword == "" {
		return errors.New("password is required")
	}
	hash, err := passwd.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.SetPassword(ctx, id, hash)
}

func (s *Service) Deactivate(ctx context.Context, id string) error { return s.repo.Deactivate(ctx, id) }
func (s *Service) Reactivate(ctx context.Context, id string) error { return s.repo.Reactivate(ctx, id) }

func (s *Service) Get(ctx context.Context, id string) (*User, error) { return s.repo.GetByID(ctx, id) }
func (s *Service) List(ctx context.Context, role *rbac.Role, active *bool) ([]*User, error) {
	return s.repo.List(ctx, role, active)
}

func (s *Service) ListSessions(ctx context.Context, userID string) ([]*Session, error) {
	return s.repo.ListSessions(ctx, userID)
}

func (s *Service) RevokeSession(ctx context.Context, userID, sessionID string) error {
	// Only revoke a session belonging to the given user.
	sess, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess.UserID != userID {
		return fmt.Errorf("session does not belong to user")
	}
	return s.repo.RevokeSession(ctx, sessionID)
}

func validateRole(r rbac.Role) error {
	switch r {
	case rbac.RoleSuperAdmin, rbac.RoleSalesDevManager, rbac.RoleSalesAgent, rbac.RoleViewer:
		return nil
	}
	return fmt.Errorf("invalid role: %s", r)
}
