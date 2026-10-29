package nextactions

import (
	"context"
	"fmt"
	"time"
)

// collectPendingReports turns every un-submitted mandatory report into a
// high-priority action so a "ثبت گزارش" never sinks beneath ordinary
// prospecting.
func (s *Service) collectPendingReports(ctx context.Context, p Params, now time.Time) ([]candidate, error) {
	q := `
		SELECT i.id, i.lead_id, i.channel, i.started_at, i.created_at,
			COALESCE(l.company_name,''), COALESCE(l.score,0),
			c.full_name, l.project_id, COALESCE(pr.title,''),
			i.user_id, COALESCE(u.name,'')
		FROM interactions i
		JOIN leads l ON l.id = i.lead_id
		LEFT JOIN contacts c ON c.id = i.contact_id
		LEFT JOIN projects pr ON pr.id = l.project_id
		LEFT JOIN users u ON u.id = i.user_id
		WHERE i.report_status = 'PENDING'
		  AND l.status NOT IN ('QUALIFIED','DISQUALIFIED','EXPORTED_TO_MUNINN','ARCHIVED')`
	args := []any{}
	if !p.Team {
		q += ` AND i.user_id = $1`
		args = append(args, p.UserID)
	}
	q += ` ORDER BY i.created_at ASC`

	rows, err := s.db.Pool().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []candidate{}
	for rows.Next() {
		var (
			id, leadID, channel, company string
			score                        int
			startedAt, createdAt         time.Time
			contactName, projectID       *string
			projectTitle, userID         string
			userName                     string
		)
		if err := rows.Scan(&id, &leadID, &channel, &startedAt, &createdAt, &company, &score,
			&contactName, &projectID, &projectTitle, &userID, &userName); err != nil {
			return nil, err
		}
		urgency, _ := urgencyFor(&createdAt, now)
		total := pendingReportBase + urgency
		reasons := []string{"گزارش تماس ثبت نشده"}
		if now.Sub(createdAt) > 24*time.Hour {
			reasons = append(reasons, "گزارش برای مدت طولانی ثبت نشده است")
		}
		out = append(out, candidate{rank: total, item: &ActionItem{
			ID:            "pending_report:" + id,
			SourceType:    SourcePendingReport,
			SourceID:      id,
			LeadID:        leadID,
			CompanyName:   company,
			LeadName:      company,
			ContactName:   contactName,
			ActionType:    ActionReport,
			Title:         "ثبت گزارش تماس با " + company,
			Reason:        "گزارش تماس ثبت نشده",
			Priority:      TierForScore(total),
			PriorityScore: total,
			DueAt:         &createdAt,
			Overdue:       now.Sub(createdAt) > 1*time.Hour,
			SuggestedAction: "report",
			CTALabel:      "ثبت گزارش",
			ProjectID:     projectID,
			ProjectTitle:  projectTitle,
			AssigneeID:    &userID,
			AssigneeName:  userName,
			Explainability: reasons,
		}})
	}
	return out, rows.Err()
}

// collectFollowUps turns pending/snoozed follow-ups into ranked actions.
func (s *Service) collectFollowUps(ctx context.Context, p Params, now time.Time) ([]candidate, error) {
	q := `
		SELECT f.id, f.lead_id, f.title, f.description, f.channel, f.type, f.source, f.due_at,
			COALESCE(l.company_name,''), COALESCE(l.score,0),
			c.full_name, f.project_id, COALESCE(pr.title,''),
			f.assigned_to, COALESCE(u.name,''),
			(SELECT max(i.started_at) FROM interactions i WHERE i.lead_id = l.id) AS last_interaction_at
		FROM follow_ups f
		JOIN leads l ON l.id = f.lead_id
		LEFT JOIN contacts c ON c.id = f.contact_id
		LEFT JOIN projects pr ON pr.id = f.project_id
		LEFT JOIN users u ON u.id = f.assigned_to
		WHERE f.status IN ('PENDING','SNOOZED')
		  AND l.status NOT IN ('QUALIFIED','DISQUALIFIED','EXPORTED_TO_MUNINN','ARCHIVED')`
	args := []any{}
	if !p.Team {
		q += ` AND f.assigned_to = $1`
		args = append(args, p.UserID)
	}
	q += ` ORDER BY f.due_at ASC`

	rows, err := s.db.Pool().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []candidate{}
	for rows.Next() {
		var (
			id, leadID, company, fuSource   string
			title, description, channel, fuType *string
			dueAt                          time.Time
			score                          int
			contactName, projectID         *string
			projectTitle, assignedTo       string
			userName                       string
			lastInteractionAt              *time.Time
		)
		if err := rows.Scan(&id, &leadID, &title, &description, &channel, &fuType, &fuSource, &dueAt,
			&company, &score, &contactName, &projectID, &projectTitle, &assignedTo, &userName, &lastInteractionAt); err != nil {
			return nil, err
		}

		urgency, urgencyReason := urgencyFor(&dueAt, now)
		quality := leadQuality(score)
		reasons := []string{}
		if urgencyReason != "" {
			reasons = append(reasons, urgencyReason)
		}
		reasons = append(reasons, qualityReason(score))

		commitment := 0
		if promisedCommitment(fuSource, strVal(title), strVal(description)) {
			commitment = commitmentExplicit
			reasons = append(reasons, "در گزارش قبلی اقدام بعدی ثبت شده")
		}

		total := urgency + quality + commitment
		actionType := actionTypeForChannel(strVal(channel), strVal(fuType))
		overdue := dueAt.Before(now)

		contextText := strVal(description)
		if contextText == "" {
			contextText = strVal(title)
		}

		titleText := "پیگیری " + company
		switch actionType {
		case ActionCall:
			titleText = "تماس با " + company
		case ActionMessage:
			titleText = "پیام به " + company
		case ActionEmail:
			titleText = "ایمیل به " + company
		}

		out = append(out, candidate{rank: total, item: &ActionItem{
			ID:            "follow_up:" + id,
			SourceType:    SourceFollowUp,
			SourceID:      id,
			LeadID:        leadID,
			CompanyName:   company,
			LeadName:      company,
			ContactName:   contactName,
			ActionType:    actionType,
			Title:         titleText,
			Reason:        reasonSummary(reasons),
			Context:       contextText,
			Priority:      TierForScore(total),
			PriorityScore: total,
			DueAt:         &dueAt,
			Overdue:       overdue,
			SuggestedAction: string(actionType),
			CTALabel:      ctaLabelFor(actionType),
			ProjectID:     projectID,
			ProjectTitle:  projectTitle,
			LastInteractionAt: lastInteractionAt,
			AssigneeID:    &assignedTo,
			AssigneeName:  userName,
			Explainability: reasons,
		}})
	}
	return out, rows.Err()
}

// collectInferredLeadActions derives lead-level actions (first contact and
// stale warm leads) that have no explicit follow-up. These are de-duplicated
// away whenever an explicit item already exists for the lead.
func (s *Service) collectInferredLeadActions(ctx context.Context, p Params, now time.Time) ([]candidate, error) {
	q := `
		SELECT l.id, l.company_name, l.status, l.score, l.project_id, COALESCE(pr.title,''), l.updated_at,
			l.owner_user_id, COALESCE(u.name,''),
			(SELECT count(*) FROM interactions i WHERE i.lead_id = l.id) AS interaction_count,
			(SELECT max(i.started_at) FROM interactions i WHERE i.lead_id = l.id) AS last_interaction_at,
			(SELECT c.full_name FROM contacts c WHERE c.lead_id = l.id
			  ORDER BY c.is_primary_contact DESC, c.is_decision_maker DESC LIMIT 1) AS contact_name
		FROM leads l
		LEFT JOIN projects pr ON pr.id = l.project_id
		LEFT JOIN users u ON u.id = l.owner_user_id
		WHERE l.status IN ('ASSIGNED','CONTACTING','FOLLOW_UP')
		  AND l.score >= 40`
	args := []any{}
	if !p.Team {
		q += ` AND l.owner_user_id = $1`
		args = append(args, p.UserID)
	}
	q += ` ORDER BY l.score DESC`

	rows, err := s.db.Pool().Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []candidate{}
	for rows.Next() {
		var (
			id, company, status, ownerID, userName string
			score, interactionCount                int
			projectID, contactName                 *string
			projectTitle                           string
			updatedAt                              time.Time
			lastInteractionAt                      *time.Time
		)
		if err := rows.Scan(&id, &company, &status, &score, &projectID, &projectTitle, &updatedAt,
			&ownerID, &userName, &interactionCount, &lastInteractionAt, &contactName); err != nil {
			return nil, err
		}

		quality := leadQuality(score)
		reason := ""
		base := 0
		actionType := ActionType("")
		title := ""
		cta := ""

		if interactionCount == 0 {
			// Newly assigned, never contacted.
			base = firstContactBase
			actionType = ActionCall
			title = "تماس اولیه با " + company
			cta = "تماس"
			reason = "تماس اولیه هنوز انجام نشده"
		} else {
			// Stale warm lead: previously contacted but idle.
			idle := time.Duration(0)
			if lastInteractionAt != nil {
				idle = now.Sub(lastInteractionAt.UTC())
			}
			const staleThreshold = 3 * 24 * time.Hour
			if idle < staleThreshold {
				continue // not stale enough to surface.
			}
			base = staleLeadBase + int(idle.Hours()/24) // up to a few points by age
			if base > staleLeadBase+10 {
				base = staleLeadBase + 10
			}
			actionType = ActionFollowUp
			title = "پیگیری مجدد " + company
			cta = "پیگیری"
			reason = "سرنخ گرم برای مدت طولانی پیگیری نشده"
		}

		total := base + quality
		reasons := []string{reason, qualityReason(score)}

		out = append(out, candidate{rank: total, item: &ActionItem{
			ID:            string(sourceForInferred(interactionCount)) + ":" + id,
			SourceType:    sourceForInferred(interactionCount),
			SourceID:      id,
			LeadID:        id,
			CompanyName:   company,
			LeadName:      company,
			ContactName:   contactName,
			ActionType:    actionType,
			Title:         title,
			Reason:        reason,
			Priority:      TierForScore(total),
			PriorityScore: total,
			SuggestedAction: string(actionType),
			CTALabel:      cta,
			ProjectID:     projectID,
			ProjectTitle:  projectTitle,
			LastInteractionAt: lastInteractionAt,
			AssigneeID:    &ownerID,
			AssigneeName:  userName,
			Explainability: reasons,
		}})
	}
	return out, rows.Err()
}

func sourceForInferred(interactionCount int) SourceType {
	if interactionCount == 0 {
		return SourceFirstContact
	}
	return SourceStaleLead
}

func reasonSummary(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	return reasons[0]
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

var _ = fmt.Sprintf
