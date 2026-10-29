package projects

import (
	"context"
	"errors"
)

// Service implements project lifecycle operations.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

var (
	ErrInvalidInput = errors.New("invalid project input")
)

// CreateParams holds the required/optional fields for project creation.
type CreateParams struct {
	Title                string
	Description          string
	ProductName          *string
	ProductDescription   *string
	TargetMarket         *string
	IdealCustomerProfile *string
	CustomerPainPoints   *string
	ValueProposition     *string
	SalesObjective       *string
	CommonObjections     *string
	CreatedBy            *string
	CampaignID           *string
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*Project, error) {
	if p.Title == "" || p.Description == "" {
		return nil, ErrInvalidInput
	}
	proj := &Project{
		Title:                p.Title,
		Description:          p.Description,
		ProductName:          p.ProductName,
		ProductDescription:   p.ProductDescription,
		TargetMarket:         p.TargetMarket,
		IdealCustomerProfile: p.IdealCustomerProfile,
		CustomerPainPoints:   p.CustomerPainPoints,
		ValueProposition:     p.ValueProposition,
		SalesObjective:       p.SalesObjective,
		CommonObjections:     p.CommonObjections,
		CreatedBy:            p.CreatedBy,
		CampaignID:           p.CampaignID,
	}
	return s.repo.Create(ctx, proj)
}

func (s *Service) Get(ctx context.Context, id string) (*Project, error) {
	return s.repo.Get(ctx, id)
}

// UpdateParams patches an existing project.
type UpdateParams struct {
	Title                string
	Description          string
	ProductName          *string
	ProductDescription   *string
	TargetMarket         *string
	IdealCustomerProfile *string
	CustomerPainPoints   *string
	ValueProposition     *string
	SalesObjective       *string
	CommonObjections     *string
	Status               *Status
	CampaignID           *string
}

func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (*Project, error) {
	proj, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	proj.Title = p.Title
	proj.Description = p.Description
	proj.ProductName = p.ProductName
	proj.ProductDescription = p.ProductDescription
	proj.TargetMarket = p.TargetMarket
	proj.IdealCustomerProfile = p.IdealCustomerProfile
	proj.CustomerPainPoints = p.CustomerPainPoints
	proj.ValueProposition = p.ValueProposition
	proj.SalesObjective = p.SalesObjective
	proj.CommonObjections = p.CommonObjections
	if p.Status != nil {
		proj.Status = *p.Status
	}
	if p.CampaignID != nil {
		proj.CampaignID = p.CampaignID
	}
	return s.repo.Update(ctx, proj)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, p ListParams) (*ListResult, error) {
	return s.repo.List(ctx, p)
}

// LeadCount returns the number of leads belonging to a project.
func (s *Service) LeadCount(ctx context.Context, projectID string) (int, error) {
	var n int
	err := s.repo.DB().Pool().QueryRow(ctx, `SELECT count(*) FROM leads WHERE project_id=$1`, projectID).Scan(&n)
	return n, err
}
