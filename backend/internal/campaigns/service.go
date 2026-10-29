package campaigns

import (
	"context"
	"errors"
	"time"
)

// Service implements campaign lifecycle operations.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

var ErrInvalidInput = errors.New("invalid campaign input")

// CreateParams holds the required/optional fields for campaign creation.
type CreateParams struct {
	Title       string
	Description *string
	Objective   *string
	StartAt     *time.Time
	EndAt       *time.Time
	CreatedBy   *string
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*Campaign, error) {
	if p.Title == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.Create(ctx, &Campaign{
		Title:       p.Title,
		Description: p.Description,
		Objective:   p.Objective,
		Status:      StatusActive,
		StartAt:     p.StartAt,
		EndAt:       p.EndAt,
		CreatedBy:   p.CreatedBy,
	})
}

func (s *Service) Get(ctx context.Context, id string) (*Campaign, error) {
	return s.repo.Get(ctx, id)
}

// UpdateParams patches an existing campaign.
type UpdateParams struct {
	Title       *string
	Description *string
	Objective   *string
	Status      *Status
	StartAt     *time.Time
	EndAt       *time.Time
}

func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (*Campaign, error) {
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Title != nil {
		if *p.Title == "" {
			return nil, ErrInvalidInput
		}
		c.Title = *p.Title
	}
	if p.Description != nil {
		c.Description = p.Description
	}
	if p.Objective != nil {
		c.Objective = p.Objective
	}
	if p.Status != nil {
		c.Status = *p.Status
	}
	if p.StartAt != nil {
		c.StartAt = p.StartAt
	}
	if p.EndAt != nil {
		c.EndAt = p.EndAt
	}
	return s.repo.Update(ctx, c)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, p ListParams) (*ListResult, error) {
	return s.repo.List(ctx, p)
}

// ListProjects returns the projects attached to a campaign.
func (s *Service) ListProjects(ctx context.Context, campaignID string) ([]*ProjectRef, error) {
	return s.repo.ListProjects(ctx, campaignID)
}
