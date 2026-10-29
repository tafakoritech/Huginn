package interactions

import (
	"context"
	"time"

	"github.com/tafakorpardazan/huginn/internal/database"
)

// Channel is a communication channel.
type Channel string

const (
	ChannelCall     Channel = "CALL"
	ChannelSMS      Channel = "SMS"
	ChannelWhatsApp Channel = "WHATSAPP"
	ChannelTelegram Channel = "TELEGRAM"
	ChannelBale     Channel = "BALE"
	ChannelEitaa    Channel = "EITAA"
	ChannelRubika   Channel = "RUBIKA"
	ChannelEmail    Channel = "EMAIL"
	ChannelOther    Channel = "OTHER"
)

var validChannels = map[Channel]bool{
	ChannelCall: true, ChannelSMS: true, ChannelWhatsApp: true, ChannelTelegram: true,
	ChannelBale: true, ChannelEitaa: true, ChannelRubika: true, ChannelEmail: true, ChannelOther: true,
}

func IsValidChannel(c Channel) bool { return validChannels[c] }

// Direction of the interaction.
type Direction string

const (
	DirectionOutbound Direction = "OUTBOUND"
	DirectionInbound  Direction = "INBOUND"
)

// ActionStatus tracks the client-side action lifecycle.
type ActionStatus string

const (
	ActionCreated   ActionStatus = "CREATED"
	ActionOpened    ActionStatus = "OPENED"
	ActionCompleted ActionStatus = "COMPLETED"
	ActionFailed    ActionStatus = "FAILED"
	ActionCancelled ActionStatus = "CANCELLED"
)

// ReportStatus tracks mandatory report state.
type ReportStatus string

const (
	ReportPending         ReportStatus = "PENDING"
	ReportSubmitted       ReportStatus = "SUBMITTED"
	ReportVoidedByManager ReportStatus = "VOIDED_BY_MANAGER"
)

// Interaction is a single communication attempt with a lead.
type Interaction struct {
	ID        string  `json:"id"`
	LeadID    string  `json:"lead_id"`
	ContactID *string `json:"contact_id"`
	UserID    string  `json:"user_id"`
	// ProjectID optionally scopes the interaction to the Project whose sales
	// context drove it. Project-wide activities may set it with lead_id set to
	// the related lead (if any).
	ProjectID           *string      `json:"project_id"`
	Channel             Channel      `json:"channel"`
	Direction           Direction    `json:"direction"`
	Objective           *string      `json:"objective"`
	StartedAt           time.Time    `json:"started_at"`
	EndedAt             *time.Time   `json:"ended_at"`
	DurationSeconds     *int         `json:"duration_seconds"`
	ActionStatus        ActionStatus `json:"action_status"`
	ReportStatus        ReportStatus `json:"report_status"`
	SuggestedMessage    *string      `json:"suggested_message"`
	SuggestedSubject    *string      `json:"suggested_subject"`
	FinalMessageIfKnown *string      `json:"final_message_if_known"`
	DeepLink            *string      `json:"deep_link"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

// Repository persists interactions.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository { return &Repository{db: db} }

const cols = `id, lead_id, contact_id, user_id, project_id, channel, direction, objective, started_at, ended_at,
	duration_seconds, action_status, report_status, suggested_message, suggested_subject,
	final_message_if_known, deep_link, created_at, updated_at`

func scan(row database.RowScanner) (*Interaction, error) {
	var i Interaction
	err := row.Scan(&i.ID, &i.LeadID, &i.ContactID, &i.UserID, &i.ProjectID, &i.Channel, &i.Direction, &i.Objective,
		&i.StartedAt, &i.EndedAt, &i.DurationSeconds, &i.ActionStatus, &i.ReportStatus,
		&i.SuggestedMessage, &i.SuggestedSubject, &i.FinalMessageIfKnown, &i.DeepLink,
		&i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *Repository) Create(ctx context.Context, i *Interaction) (*Interaction, error) {
	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO interactions (lead_id, contact_id, user_id, project_id, channel, direction, objective,
			action_status, report_status, suggested_message, suggested_subject, deep_link)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING `+cols,
		i.LeadID, i.ContactID, i.UserID, i.ProjectID, i.Channel, i.Direction, i.Objective,
		i.ActionStatus, i.ReportStatus, i.SuggestedMessage, i.SuggestedSubject, i.DeepLink)
	return scan(row)
}

func (r *Repository) Get(ctx context.Context, id string) (*Interaction, error) {
	row := r.db.Pool().QueryRow(ctx, `SELECT `+cols+` FROM interactions WHERE id=$1`, id)
	return scan(row)
}

func (r *Repository) UpdateActionStatus(ctx context.Context, id string, status ActionStatus, endedAt *time.Time, duration *int) error {
	_, err := r.db.Pool().Exec(ctx, `
		UPDATE interactions SET action_status=$2, ended_at=$3, duration_seconds=$4, updated_at=now() WHERE id=$1`,
		id, status, endedAt, duration)
	return err
}

func (r *Repository) UpdateReportStatus(ctx context.Context, id string, status ReportStatus) error {
	_, err := r.db.Pool().Exec(ctx, `UPDATE interactions SET report_status=$2, updated_at=now() WHERE id=$1`, id, status)
	return err
}

// PendingReportForUser returns the earliest pending report for a user.
func (r *Repository) PendingReportForUser(ctx context.Context, userID string) (*Interaction, error) {
	row := r.db.Pool().QueryRow(ctx, `
		SELECT `+cols+` FROM interactions
		WHERE user_id=$1 AND report_status='PENDING'
		ORDER BY created_at ASC LIMIT 1`, userID)
	i, err := scan(row)
	if err != nil {
		if database.IsNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return i, nil
}

func (r *Repository) CountPendingReportsForUser(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.Pool().QueryRow(ctx, `
		SELECT count(*) FROM interactions WHERE user_id=$1 AND report_status='PENDING'`, userID).Scan(&n)
	return n, err
}

func (r *Repository) CountPendingReportsAll(ctx context.Context) (int, error) {
	var n int
	err := r.db.Pool().QueryRow(ctx, `SELECT count(*) FROM interactions WHERE report_status='PENDING'`).Scan(&n)
	return n, err
}

func (r *Repository) ListForLead(ctx context.Context, leadID string) ([]*Interaction, error) {
	rows, err := r.db.Pool().Query(ctx, `SELECT `+cols+` FROM interactions WHERE lead_id=$1 ORDER BY created_at DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Interaction{}
	for rows.Next() {
		i, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// CountCallsToday counts outbound calls started today for a user (or all if userID empty).
func (r *Repository) CountCallsToday(ctx context.Context, userID string) (int, error) {
	q := `SELECT count(*) FROM interactions WHERE channel='CALL' AND started_at >= date_trunc('day', now())`
	args := []any{}
	if userID != "" {
		q += ` AND user_id=$1`
		args = append(args, userID)
	}
	var n int
	err := r.db.Pool().QueryRow(ctx, q, args...).Scan(&n)
	return n, err
}

func (r *Repository) CountByChannelAndOutcome(ctx context.Context, channel Channel) (int, error) {
	var n int
	err := r.db.Pool().QueryRow(ctx, `
		SELECT count(*) FROM interactions i
		WHERE i.channel=$1 AND i.started_at >= date_trunc('day', now())`, channel).Scan(&n)
	return n, err
}
