package leads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/contacts"
	"github.com/tafakorpardazan/huginn/internal/database"
	"github.com/tafakorpardazan/huginn/internal/leadversions"
)

// Service implements lead lifecycle operations with strict state enforcement
// and immutable, versioned editing. When the optional contacts and versions
// dependencies are provided, every mutation appends a new immutable snapshot
// to lead_versions while preserving all previous state.
type Service struct {
	repo     *Repository
	contacts *contacts.Repository
	versions *leadversions.Repository
}

// Deps are optional dependencies that enable versioned editing.
type Deps struct {
	Contacts *contacts.Repository
	Versions *leadversions.Repository
}

func NewService(repo *Repository, deps ...Deps) *Service {
	s := &Service{repo: repo}
	if len(deps) > 0 {
		s.contacts = deps[0].Contacts
		s.versions = deps[0].Versions
	}
	return s
}

var (
	ErrNotFound     = errors.New("lead not found")
	ErrInvalidState = errors.New("invalid state transition")
)

type CreateParams struct {
	CompanyName      string
	DisplayName      *string
	BusinessType     *string
	Industry         *string
	Description      *string
	Province         *string
	City             *string
	Address          *string
	Website          *string
	Instagram        *string
	Source           *string
	SourceBatchID    *string
	SourceDocumentID *string
	ProjectID        *string
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*Lead, error) {
	if p.CompanyName == "" {
		return nil, errors.New("company_name is required")
	}
	// project_id is optional at creation: leads imported from documents are
	// created without a project and may be linked to one later via Update.
	l := &Lead{
		CompanyName:      p.CompanyName,
		DisplayName:      p.DisplayName,
		BusinessType:     p.BusinessType,
		Industry:         p.Industry,
		Description:      p.Description,
		Province:         p.Province,
		City:             p.City,
		Address:          p.Address,
		Website:          p.Website,
		Instagram:        p.Instagram,
		Source:           p.Source,
		SourceBatchID:    p.SourceBatchID,
		SourceDocumentID: p.SourceDocumentID,
		ProjectID:        p.ProjectID,
		Status:           StatusNew,
		QualityStatus:    QualityRaw,
	}
	created, err := s.repo.Create(ctx, l)
	if err != nil {
		return nil, err
	}
	if s.versions != nil {
		// Initial immutable snapshot (version 1) so admins see the creation
		// in the change history. Runs on its own statement; creation is
		// brand-new so there is nothing to race.
		if _, err := s.versions.RecordTx(ctx, s.repo.DB().Pool(), created.ID, nil, s.buildSnapshot(created, nil, nil), nil, nil, nil); err != nil {
			return nil, err
		}
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Lead, error) {
	l, err := s.repo.Get(ctx, id)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	tags, err := s.repo.TagsForLead(ctx, l.ID)
	if err == nil {
		l.Tags = tags
	}
	return l, nil
}

// ---- Versioned editing ----

// leadFieldSetters maps a JSON field name to a setter on the lead. A nil value
// clears the field (null); an absent key is ignored.
var leadFieldSetters = map[string]func(*Lead, *string){
	"company_name": func(l *Lead, v *string) {
		if v != nil {
			l.CompanyName = *v
		} else {
			l.CompanyName = ""
		}
	},
	"display_name":       func(l *Lead, v *string) { l.DisplayName = v },
	"business_type":      func(l *Lead, v *string) { l.BusinessType = v },
	"industry":           func(l *Lead, v *string) { l.Industry = v },
	"description":        func(l *Lead, v *string) { l.Description = v },
	"province":           func(l *Lead, v *string) { l.Province = v },
	"city":               func(l *Lead, v *string) { l.City = v },
	"address":            func(l *Lead, v *string) { l.Address = v },
	"website":            func(l *Lead, v *string) { l.Website = v },
	"instagram":          func(l *Lead, v *string) { l.Instagram = v },
	"source":             func(l *Lead, v *string) { l.Source = v },
	"source_batch_id":    func(l *Lead, v *string) { l.SourceBatchID = v },
	"source_document_id": func(l *Lead, v *string) { l.SourceDocumentID = v },
	"project_id":         func(l *Lead, v *string) { l.ProjectID = v },
}

// Update patches lead fields versioned: it appends a new immutable snapshot
// capturing the old and new values for every changed field. patches maps JSON
// field names to *string values; a nil value clears the field.
func (s *Service) Update(ctx context.Context, leadID string, patches map[string]*string, actorID, note string) (*Lead, error) {
	if len(patches) == 0 {
		return s.Get(ctx, leadID)
	}
	err := s.repo.DB().WithTx(ctx, func(q database.Querier) error {
		l, err := s.repo.GetForUpdate(ctx, q, leadID)
		if err != nil {
			if database.IsNoRows(err) {
				return ErrNotFound
			}
			return err
		}
		contactsList, err := s.listContactsTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		tags, err := s.repo.TagsForLeadTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldSnap := s.buildSnapshot(l, contactsList, tags)

		// A status patch is a validated state transition within the same
		// versioned edit.
		if sv, ok := patches["status"]; ok {
			delete(patches, "status")
			if sv != nil {
				to := Status(*sv)
				if !IsValidStatus(to) {
					return errors.New("invalid status")
				}
				if !CanTransition(l.Status, to) {
					return &ErrInvalidTransition{From: l.Status, To: to}
				}
				if l.Status != to {
					from := l.Status
					l.Status = to
					if err := s.repo.InsertStatusHistoryTx(ctx, q, leadID, from, to, note, strptr(actorID)); err != nil {
						return err
					}
				}
			}
		}

		for k, v := range patches {
			if setter, ok := leadFieldSetters[k]; ok {
				setter(l, v)
			}
		}
		if err := s.repo.UpdateTx(ctx, q, l); err != nil {
			return err
		}
		newSnap := s.buildSnapshot(l, contactsList, tags)
		if err := s.recordVersion(ctx, q, leadID, oldSnap, newSnap, actorID, note); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, leadID)
}

// CreateContact adds a contact to a lead and records a new version.
func (s *Service) CreateContact(ctx context.Context, leadID string, c *contacts.Contact, actorID, note string) (*contacts.Contact, error) {
	c.LeadID = leadID
	var result *contacts.Contact
	err := s.repo.DB().WithTx(ctx, func(q database.Querier) error {
		l, err := s.repo.GetForUpdate(ctx, q, leadID)
		if err != nil {
			if database.IsNoRows(err) {
				return ErrNotFound
			}
			return err
		}
		contactsList, err := s.listContactsTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		tags, err := s.repo.TagsForLeadTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldSnap := s.buildSnapshot(l, contactsList, tags)

		created, err := s.contacts.CreateTx(ctx, q, c)
		if err != nil {
			return err
		}
		contactsList = append(contactsList, created)
		if err := s.recordVersion(ctx, q, leadID, oldSnap, s.buildSnapshot(l, contactsList, tags), actorID, note); err != nil {
			return err
		}
		result = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// contactFieldSetters maps a JSON field name to a setter on a contact. A nil
// *string clears the field; a nil *bool keeps the current value (JSON null for
// a boolean field is treated as "not provided" to avoid destroying flags).
var contactStringSetters = map[string]func(*contacts.Contact, *string){
	"first_name": func(c *contacts.Contact, v *string) { c.FirstName = v },
	"last_name":  func(c *contacts.Contact, v *string) { c.LastName = v },
	"full_name":  func(c *contacts.Contact, v *string) { c.FullName = v },
	"job_title":  func(c *contacts.Contact, v *string) { c.JobTitle = v },
	"department": func(c *contacts.Contact, v *string) { c.Department = v },
	"mobile":     func(c *contacts.Contact, v *string) { c.Mobile = v },
	"phone":      func(c *contacts.Contact, v *string) { c.Phone = v },
	"email":      func(c *contacts.Contact, v *string) { c.Email = v },
	"whatsapp":   func(c *contacts.Contact, v *string) { c.WhatsApp = v },
	"telegram":   func(c *contacts.Contact, v *string) { c.Telegram = v },
	"bale":       func(c *contacts.Contact, v *string) { c.Bale = v },
	"eitaa":      func(c *contacts.Contact, v *string) { c.Eitaa = v },
	"rubika":     func(c *contacts.Contact, v *string) { c.Rubika = v },
}

// UpdateContact patches a contact on a lead and records a new version.
// patches maps JSON field names to *string or *bool values.
func (s *Service) UpdateContact(ctx context.Context, leadID, contactID string, patches map[string]any, actorID, note string) (*contacts.Contact, error) {
	if len(patches) == 0 {
		return nil, errors.New("no fields to update")
	}
	var result *contacts.Contact
	err := s.repo.DB().WithTx(ctx, func(q database.Querier) error {
		l, err := s.repo.GetForUpdate(ctx, q, leadID)
		if err != nil {
			if database.IsNoRows(err) {
				return ErrNotFound
			}
			return err
		}
		contactsList, err := s.listContactsTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		tags, err := s.repo.TagsForLeadTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldSnap := s.buildSnapshot(l, contactsList, tags)

		var target *contacts.Contact
		for _, c := range contactsList {
			if c.ID == contactID {
				target = c
				break
			}
		}
		if target == nil {
			return errors.New("contact not found")
		}
		for k, v := range patches {
			if setter, ok := contactStringSetters[k]; ok {
				if sv, ok := v.(*string); ok {
					setter(target, sv)
				}
				continue
			}
			switch k {
			case "is_decision_maker":
				if b, ok := v.(*bool); ok {
					target.IsDecisionMaker = *b
				}
			case "is_primary_contact":
				if b, ok := v.(*bool); ok {
					target.IsPrimaryContact = *b
				}
			}
		}
		if err := s.contacts.UpdateTx(ctx, q, target); err != nil {
			return err
		}
		if err := s.recordVersion(ctx, q, leadID, oldSnap, s.buildSnapshot(l, contactsList, tags), actorID, note); err != nil {
			return err
		}
		result = target
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteContact soft-deletes a contact and records a new version. Historical
// versions keep the contact data.
func (s *Service) DeleteContact(ctx context.Context, leadID, contactID, actorID, note string) error {
	return s.repo.DB().WithTx(ctx, func(q database.Querier) error {
		l, err := s.repo.GetForUpdate(ctx, q, leadID)
		if err != nil {
			if database.IsNoRows(err) {
				return ErrNotFound
			}
			return err
		}
		contactsList, err := s.listContactsTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		tags, err := s.repo.TagsForLeadTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldSnap := s.buildSnapshot(l, contactsList, tags)

		found := false
		for _, c := range contactsList {
			if c.ID == contactID {
				found = true
				break
			}
		}
		if !found {
			return errors.New("contact not found")
		}
		if err := s.contacts.SoftDeleteTx(ctx, q, contactID, leadID); err != nil {
			return err
		}
		remaining, err := s.listContactsTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		return s.recordVersion(ctx, q, leadID, oldSnap, s.buildSnapshot(l, remaining, tags), actorID, note)
	})
}

// AddTag attaches a tag and records a new version when the set actually changes.
func (s *Service) AddTag(ctx context.Context, leadID, tagName, source, actorID, note string) error {
	return s.repo.DB().WithTx(ctx, func(q database.Querier) error {
		l, err := s.repo.GetForUpdate(ctx, q, leadID)
		if err != nil {
			if database.IsNoRows(err) {
				return ErrNotFound
			}
			return err
		}
		contactsList, err := s.listContactsTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldTags, err := s.repo.TagsForLeadTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldSnap := s.buildSnapshot(l, contactsList, oldTags)

		added, err := s.repo.AddTagTx(ctx, q, leadID, tagName, source)
		if err != nil {
			return err
		}
		if !added {
			return nil
		}
		newTags, err := s.repo.TagsForLeadTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		return s.recordVersion(ctx, q, leadID, oldSnap, s.buildSnapshot(l, contactsList, newTags), actorID, note)
	})
}

// RemoveTag detaches a tag and records a new version when the set changes.
func (s *Service) RemoveTag(ctx context.Context, leadID, tagName, actorID, note string) error {
	return s.repo.DB().WithTx(ctx, func(q database.Querier) error {
		l, err := s.repo.GetForUpdate(ctx, q, leadID)
		if err != nil {
			if database.IsNoRows(err) {
				return ErrNotFound
			}
			return err
		}
		contactsList, err := s.listContactsTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldTags, err := s.repo.TagsForLeadTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldSnap := s.buildSnapshot(l, contactsList, oldTags)

		removed, err := s.repo.RemoveTagTx(ctx, q, leadID, tagName)
		if err != nil {
			return err
		}
		if !removed {
			return nil
		}
		newTags, err := s.repo.TagsForLeadTx(ctx, q, leadID)
		if err != nil {
			return err
		}
		return s.recordVersion(ctx, q, leadID, oldSnap, s.buildSnapshot(l, contactsList, newTags), actorID, note)
	})
}

// ListVersions returns the immutable change history for a lead, newest first.
func (s *Service) ListVersions(ctx context.Context, leadID string, limit, offset int) ([]*leadversions.Version, error) {
	if s.versions == nil {
		return nil, nil
	}
	return s.versions.List(ctx, leadID, limit, offset)
}

// GetVersion returns a single immutable snapshot of a lead.
func (s *Service) GetVersion(ctx context.Context, leadID, versionID string) (*leadversions.Version, error) {
	if s.versions == nil {
		return nil, ErrNotFound
	}
	return s.versions.Get(ctx, leadID, versionID)
}

// RecentChanges returns the latest versions across all leads for the admin
// change-history dashboard.
func (s *Service) RecentChanges(ctx context.Context, limit, offset int) ([]*leadversions.RecentItem, error) {
	if s.versions == nil {
		return nil, nil
	}
	return s.versions.Recent(ctx, limit, offset)
}

// CountVersions returns how many versions a lead has.
func (s *Service) CountVersions(ctx context.Context, leadID string) (int, error) {
	if s.versions == nil {
		return 0, nil
	}
	return s.versions.Count(ctx, leadID)
}

// ---- Versioned state transitions ----

// ChangeStatus validates and performs a state transition atomically, recording
// a new version when versioning is enabled.
func (s *Service) ChangeStatus(ctx context.Context, leadID string, to Status, reason string, actorID string) (*Lead, error) {
	var result *Lead
	err := s.repo.DB().WithTx(ctx, func(q database.Querier) error {
		l, err := s.repo.GetForUpdate(ctx, q, leadID)
		if err != nil {
			if database.IsNoRows(err) {
				return ErrNotFound
			}
			return err
		}
		if !CanTransition(l.Status, to) {
			return &ErrInvalidTransition{From: l.Status, To: to}
		}
		if l.Status == to {
			result = l
			return nil
		}
		oldSnap, err := s.snapshotFor(ctx, q, l)
		if err != nil {
			return err
		}
		if err := s.repo.UpdateStatusTx(ctx, q, leadID, to); err != nil {
			return err
		}
		if err := s.repo.InsertStatusHistoryTx(ctx, q, leadID, l.Status, to, reason, strptr(actorID)); err != nil {
			return err
		}
		l.Status = to
		newSnap, err := s.snapshotFor(ctx, q, l)
		if err != nil {
			return err
		}
		if err := s.recordVersion(ctx, q, leadID, oldSnap, newSnap, actorID, reason); err != nil {
			return err
		}
		result = l
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Assign assigns a lead to a user and records a new version when ownership or
// status changes.
func (s *Service) Assign(ctx context.Context, leadID, assigneeID, assignedBy, method string) (*Lead, error) {
	if method == "" {
		method = "manual"
	}
	// validate assignee exists
	if err := s.repo.DB().Pool().QueryRow(ctx, `SELECT 1 FROM users WHERE id=$1 AND is_active=true`, assigneeID).Scan(new(int)); err != nil {
		return nil, errors.New("assignee not found or inactive")
	}

	err := s.repo.DB().WithTx(ctx, func(q database.Querier) error {
		l, err := s.repo.GetForUpdate(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldSnap, err := s.snapshotFor(ctx, q, l)
		if err != nil {
			return err
		}
		// record assignment history (idempotent per unique triple)
		_, err = q.Exec(ctx, `
			INSERT INTO assignments (lead_id, assignee_id, assigned_by, method)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (lead_id, assignee_id, method) DO UPDATE SET method=EXCLUDED.method`,
			leadID, assigneeID, strptr(assignedBy), method)
		if err != nil {
			return err
		}
		if err := s.repo.SetOwnerTx(ctx, q, leadID, assigneeID); err != nil {
			return err
		}
		// only advance state for pipeline-progressing statuses
		switch l.Status {
		case StatusNew, StatusReady, StatusDisqualified:
			if err := s.repo.UpdateStatusTx(ctx, q, leadID, StatusAssigned); err != nil {
				return err
			}
			if err := s.repo.InsertStatusHistoryTx(ctx, q, leadID, l.Status, StatusAssigned, "assigned", strptr(assignedBy)); err != nil {
				return err
			}
			l.Status = StatusAssigned
		}
		l.OwnerUserID = &assigneeID
		newSnap, err := s.snapshotFor(ctx, q, l)
		if err != nil {
			return err
		}
		return s.recordVersion(ctx, q, leadID, oldSnap, newSnap, assignedBy, "assigned")
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, leadID)
}

func (r *Repository) SetOwnerTx(ctx context.Context, q database.Querier, id string, ownerID string) error {
	_, err := q.Exec(ctx, `UPDATE leads SET owner_user_id=$2, updated_at=now() WHERE id=$1`, id, ownerID)
	return err
}

// Unassign removes the owner and records a new version when the state changes.
func (s *Service) Unassign(ctx context.Context, leadID string) (*Lead, error) {
	err := s.repo.DB().WithTx(ctx, func(q database.Querier) error {
		l, err := s.repo.GetForUpdate(ctx, q, leadID)
		if err != nil {
			return err
		}
		oldSnap, err := s.snapshotFor(ctx, q, l)
		if err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `UPDATE leads SET owner_user_id=NULL, updated_at=now() WHERE id=$1`, leadID); err != nil {
			return err
		}
		l.OwnerUserID = nil
		if l.Status == StatusAssigned {
			if err := s.repo.UpdateStatusTx(ctx, q, leadID, StatusReady); err != nil {
				return err
			}
			if err := s.repo.InsertStatusHistoryTx(ctx, q, leadID, StatusAssigned, StatusReady, "unassigned", nil); err != nil {
				return err
			}
			l.Status = StatusReady
		}
		newSnap, err := s.snapshotFor(ctx, q, l)
		if err != nil {
			return err
		}
		return s.recordVersion(ctx, q, leadID, oldSnap, newSnap, "", "unassigned")
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, leadID)
}

// Archive archives a lead.
func (s *Service) Archive(ctx context.Context, leadID, actorID, reason string) (*Lead, error) {
	_, err := s.ChangeStatus(ctx, leadID, StatusArchived, reason, actorID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, leadID)
}

// AssignRoundRobin assigns a batch of leads round-robin to a list of agents.
func (s *Service) AssignRoundRobin(ctx context.Context, leadIDs, agentIDs []string, assignedBy string) (int, error) {
	if len(agentIDs) == 0 {
		return 0, errors.New("no agents provided")
	}
	count := 0
	for i, id := range leadIDs {
		agent := agentIDs[i%len(agentIDs)]
		if _, err := s.Assign(ctx, id, agent, assignedBy, "round_robin"); err == nil {
			count++
		}
	}
	return count, nil
}

// List delegates to repository.
func (s *Service) List(ctx context.Context, p ListParams, viewerRole, viewerID string) (*ListResult, error) {
	return s.repo.List(ctx, p, viewerRole, viewerID)
}

// WorkQueue delegates to repository.
func (s *Service) WorkQueue(ctx context.Context, userID string) ([]*WorkQueueItem, error) {
	return s.repo.WorkQueue(ctx, userID)
}

// ---- snapshot helpers ----

func (s *Service) listContactsTx(ctx context.Context, q database.Querier, leadID string) ([]*contacts.Contact, error) {
	if s.contacts == nil {
		return nil, nil
	}
	return s.contacts.ListForLeadTx(ctx, q, leadID)
}

// snapshotFor builds a full snapshot from a locked lead using the tx.
func (s *Service) snapshotFor(ctx context.Context, q database.Querier, l *Lead) (*leadversions.Snapshot, error) {
	contactsList, err := s.listContactsTx(ctx, q, l.ID)
	if err != nil {
		return nil, err
	}
	tags, err := s.repo.TagsForLeadTx(ctx, q, l.ID)
	if err != nil {
		return nil, err
	}
	return s.buildSnapshot(l, contactsList, tags), nil
}

func (s *Service) buildSnapshot(l *Lead, contactsList []*contacts.Contact, tags []string) *leadversions.Snapshot {
	snap := &leadversions.Snapshot{
		Lead: map[string]any{
			"company_name":       l.CompanyName,
			"display_name":       nilVal(l.DisplayName),
			"business_type":      nilVal(l.BusinessType),
			"industry":           nilVal(l.Industry),
			"description":        nilVal(l.Description),
			"province":           nilVal(l.Province),
			"city":               nilVal(l.City),
			"address":            nilVal(l.Address),
			"website":            nilVal(l.Website),
			"instagram":          nilVal(l.Instagram),
			"source":             nilVal(l.Source),
			"source_batch_id":    nilVal(l.SourceBatchID),
			"source_document_id": nilVal(l.SourceDocumentID),
			"status":             string(l.Status),
			"quality_status":     string(l.QualityStatus),
			"score":              l.Score,
			"owner_user_id":      nilVal(l.OwnerUserID),
			"project_id":         nilVal(l.ProjectID),
			"archived_at":        timeOrNil(l.ArchivedAt),
		},
		Tags: tags,
	}
	if snap.Tags == nil {
		snap.Tags = []string{}
	}
	if len(contactsList) == 0 {
		snap.Contacts = []map[string]any{}
	}
	for _, c := range contactsList {
		snap.Contacts = append(snap.Contacts, map[string]any{
			"id":                 c.ID,
			"first_name":         nilVal(c.FirstName),
			"last_name":          nilVal(c.LastName),
			"full_name":          nilVal(c.FullName),
			"job_title":          nilVal(c.JobTitle),
			"department":         nilVal(c.Department),
			"mobile":             nilVal(c.Mobile),
			"phone":              nilVal(c.Phone),
			"email":              nilVal(c.Email),
			"whatsapp":           nilVal(c.WhatsApp),
			"telegram":           nilVal(c.Telegram),
			"bale":               nilVal(c.Bale),
			"eitaa":              nilVal(c.Eitaa),
			"rubika":             nilVal(c.Rubika),
			"is_decision_maker":  c.IsDecisionMaker,
			"is_primary_contact": c.IsPrimaryContact,
		})
	}
	return snap
}

// recordVersion appends a version when versioning is enabled and there is an
// actual change.
func (s *Service) recordVersion(ctx context.Context, q database.Querier, leadID string, oldSnap, newSnap *leadversions.Snapshot, actorID, note string) error {
	if s.versions == nil {
		return nil
	}
	if !leadversions.HasChanges(oldSnap, newSnap) {
		return nil
	}
	_, err := s.versions.RecordTx(ctx, q, leadID, oldSnap, newSnap, strptr(actorID), s.actorRole(ctx, q, actorID), strptr(note))
	return err
}

func (s *Service) actorRole(ctx context.Context, q database.Querier, actorID string) *string {
	if actorID == "" {
		return nil
	}
	// Guard against non-UUID actor ids so an invalid reference never aborts the
	// surrounding transaction.
	if _, err := uuid.Parse(actorID); err != nil {
		return nil
	}
	var role string
	if err := q.QueryRow(ctx, `SELECT role::text FROM users WHERE id=$1`, actorID).Scan(&role); err != nil {
		return nil
	}
	return &role
}

func strptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilVal[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

func timeOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

var _ = fmt.Sprintf
var _ = json.Marshal
