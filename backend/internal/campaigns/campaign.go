// Package campaigns implements the Campaign domain entity. A Campaign is a
// time-bound / strategic sales initiative that groups one or more Projects
// (projects.campaign_id → campaigns.id). Campaigns provide the top-level
// organization above projects; each project keeps its own leads and AI context.
package campaigns

import (
	"time"

	"github.com/tafakorpardazan/huginn/internal/rbac"
)

// Status is a campaign lifecycle state.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusArchived Status = "ARCHIVED"
)

var AllStatuses = []Status{StatusActive, StatusArchived}

// Campaign is a strategic sales campaign grouping Projects.
type Campaign struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Objective   *string    `json:"objective,omitempty"`
	Status      Status     `json:"status"`
	StartAt     *time.Time `json:"start_at,omitempty"`
	EndAt       *time.Time `json:"end_at,omitempty"`
	CreatedBy   *string    `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ProjectRef is a lightweight Project reference nested in a campaign detail
// response so the dashboard can render the campaign's projects.
type ProjectRef struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Role alias for convenient permission checks from the campaigns package.
type Role = rbac.Role
