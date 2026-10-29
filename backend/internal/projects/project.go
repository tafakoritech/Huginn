// Package projects implements the Project domain entity. A Project represents a
// sales initiative — a product, campaign, service, or business objective — that
// provides the business context scoping leads and driving AI sales-scenario
// generation. A project owns many leads (1..*).
package projects

import (
	"time"

	"github.com/tafakorpardazan/huginn/internal/rbac"
)

// Status is a project lifecycle state.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusArchived Status = "ARCHIVED"
)

var AllStatuses = []Status{StatusActive, StatusArchived}

// Project is a sales initiative that groups leads and supplies AI context.
type Project struct {
	ID                   string  `json:"id"`
	Title                string  `json:"title"`
	Description          string  `json:"description"`
	ProductName          *string `json:"product_name,omitempty"`
	ProductDescription   *string `json:"product_description,omitempty"`
	TargetMarket         *string `json:"target_market,omitempty"`
	IdealCustomerProfile *string `json:"ideal_customer_profile,omitempty"`
	CustomerPainPoints   *string `json:"customer_pain_points,omitempty"`
	ValueProposition     *string `json:"value_proposition,omitempty"`
	SalesObjective       *string `json:"sales_objective,omitempty"`
	CommonObjections     *string `json:"common_objections,omitempty"`
	CreatedBy            *string `json:"created_by,omitempty"`
	Status               Status  `json:"status"`
	// CampaignID optionally groups the project under a higher-level Campaign.
	CampaignID *string   `json:"campaign_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Role alias for convenient permission checks from the project package.
type Role = rbac.Role

// LeadCount is returned alongside projects in list responses.
type LeadCount struct {
	ProjectID string
	Count     int
}

// ProductInformation composes the product_name and product_description fields
// into a single human-readable string suitable for LLM context.
func (p *Project) ProductInformation() string {
	parts := []string{}
	if p.ProductName != nil && *p.ProductName != "" {
		parts = append(parts, *p.ProductName)
	}
	if p.ProductDescription != nil && *p.ProductDescription != "" {
		parts = append(parts, *p.ProductDescription)
	}
	return joinNonEmpty(" — ", parts)
}

// TargetCustomer composes the target_market and ideal_customer_profile fields.
func (p *Project) TargetCustomer() string {
	parts := []string{}
	if p.TargetMarket != nil && *p.TargetMarket != "" {
		parts = append(parts, *p.TargetMarket)
	}
	if p.IdealCustomerProfile != nil && *p.IdealCustomerProfile != "" {
		parts = append(parts, *p.IdealCustomerProfile)
	}
	return joinNonEmpty("؛ ", parts)
}

func joinNonEmpty(sep string, parts []string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += sep
		}
		out += p
	}
	return out
}
