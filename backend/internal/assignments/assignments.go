package assignments

import (
	"context"
	"errors"

	"github.com/tafakorpardazan/huginn/internal/database"
)

// Assignment is a record of an assignment event.
type Assignment struct {
	ID         string `json:"id"`
	LeadID     string `json:"lead_id"`
	AssigneeID string `json:"assignee_id"`
	AssignedBy *string `json:"assigned_by"`
	Method     string `json:"method"`
	CreatedAt  string `json:"created_at"`
}

// Repository persists assignment history.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) HistoryForLead(ctx context.Context, leadID string) ([]*Assignment, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id, lead_id, assignee_id, assigned_by, method, created_at::text
		FROM assignments WHERE lead_id=$1 ORDER BY created_at DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Assignment{}
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.LeadID, &a.AssigneeID, &a.AssignedBy, &a.Method, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

// Service provides assignment business operations (round-robin, bulk).
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

var ErrNoAgents = errors.New("no agents provided")

// RoundRobinAgents returns the active agents to rotate across.
func (r *Repository) RoundRobinAgents(ctx context.Context) ([]string, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT id FROM users WHERE role IN ('sales_agent') AND is_active=true ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
