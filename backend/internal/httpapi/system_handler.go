package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tafakorpardazan/huginn/internal/auth"
	"github.com/tafakorpardazan/huginn/internal/cache"
	"github.com/tafakorpardazan/huginn/internal/database"
)

type systemHandler struct {
	db    *database.DB
	cache *cache.Client
}

func newSystemHandler(db *database.DB, cache *cache.Client) *systemHandler {
	return &systemHandler{db: db, cache: cache}
}

func (h *systemHandler) PendingJobs(w http.ResponseWriter, r *http.Request) {
	var n int
	err := h.db.Pool().QueryRow(r.Context(), `
		SELECT count(*) FROM import_batches WHERE status IN ('QUEUED','PROCESSING')`).Scan(&n)
	if err != nil {
		internalErr(w, err)
		return
	}
	RespondOK(w, map[string]any{"count": n})
}

type notificationItem struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Link      string    `json:"link"`
	CreatedAt time.Time `json:"created_at"`
	Read      bool      `json:"read"`
}

// Notifications computes a live notification feed for the user.
func (h *systemHandler) Notifications(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	ctx := r.Context()
	items := []notificationItem{}
	now := time.Now()

	lastRead := time.Time{}
	if h.cache != nil {
		if v, err := h.cache.Get(ctx, "notif:read:"+p.UserID); err == nil {
			if ts, err := time.Parse(time.RFC3339, v); err == nil {
				lastRead = ts
			}
		}
	}
	markRead := func(t time.Time) bool { return !t.After(lastRead) }

	// 1-2. Batch processing completed / failed.
	rows, err := h.db.Pool().Query(ctx, `
		SELECT id, name, status, COALESCE(processing_finished_at, created_at)
		FROM import_batches
		WHERE status IN ('COMPLETED','FAILED') AND created_at >= now() - interval '2 days'
		ORDER BY created_at DESC LIMIT 8`)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, status string
		var at time.Time
		if err := rows.Scan(&id, &name, &status, &at); err != nil {
			continue
		}
		nType := "batch_completed"
		title := "پردازش دسته تکمیل شد"
		if status == "FAILED" {
			nType = "batch_failed"
			title = "پردازش دسته ناموفق بود"
		}
		items = append(items, notificationItem{
			ID: nType + ":" + id, Type: nType, Title: title, Body: name,
			Link: "/imports/" + id, CreatedAt: at, Read: markRead(at),
		})
	}

	// 3. Duplicate review required.
	var dupCount int
	if err := h.db.Pool().QueryRow(ctx, `SELECT count(*) FROM possible_duplicates WHERE status='POSSIBLE'`).Scan(&dupCount); err == nil && dupCount > 0 {
		items = append(items, notificationItem{
			ID: "duplicates_review", Type: "duplicates_review",
			Title: "نیاز به بررسی تکراری‌ها", Body: strconv.Itoa(dupCount) + " مورد تکراری در انتظار بررسی",
			Link: "/duplicates", CreatedAt: now, Read: markRead(now),
		})
	}

	// 4. Pending contact reports.
	var pendingReports int
	if err := h.db.Pool().QueryRow(ctx, `SELECT count(*) FROM interactions WHERE report_status='PENDING'`).Scan(&pendingReports); err == nil && pendingReports > 0 {
		items = append(items, notificationItem{
			ID: "pending_reports", Type: "pending_reports",
			Title: "گزارش تماس در انتظار ثبت", Body: strconv.Itoa(pendingReports) + " گزارش در انتظار ثبت",
			Link: "/reports", CreatedAt: now, Read: markRead(now),
		})
	}

	// 5. Overdue follow-ups.
	var overdueFu int
	if err := h.db.Pool().QueryRow(ctx, `SELECT count(*) FROM follow_ups WHERE status='OVERDUE'`).Scan(&overdueFu); err == nil && overdueFu > 0 {
		items = append(items, notificationItem{
			ID: "overdue_followups", Type: "overdue_followups",
			Title: "پیگیری عقب‌افتاده", Body: strconv.Itoa(overdueFu) + " پیگیری از موعد گذشته است",
			Link: "/followups", CreatedAt: now, Read: markRead(now),
		})
	}

	// 6. Muninn export failures.
	erows, err := h.db.Pool().Query(ctx, `
		SELECT id, huginn_lead_id, last_error, COALESCE(exported_at, updated_at)
		FROM crm_exports WHERE status='FAILED' AND updated_at >= now() - interval '2 days'
		ORDER BY updated_at DESC LIMIT 5`)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer erows.Close()
	for erows.Next() {
		var id, leadID string
		var lastErr *string
		var at time.Time
		if err := erows.Scan(&id, &leadID, &lastErr, &at); err != nil {
			continue
		}
		body := "خطا در انتقال به Muninn"
		if lastErr != nil && *lastErr != "" {
			body = *lastErr
		}
		items = append(items, notificationItem{
			ID: "muninn_export_failure:" + id, Type: "muninn_export_failure",
			Title: "خطا در انتقال به Muninn", Body: body,
			Link: "/leads/" + leadID, CreatedAt: at, Read: markRead(at),
		})
	}

	// 7. High-value qualified leads.
	qrows, err := h.db.Pool().Query(ctx, `
		SELECT id, company_name, score, updated_at FROM leads
		WHERE status='QUALIFIED' AND score >= 80 AND updated_at >= now() - interval '1 day'
		ORDER BY updated_at DESC LIMIT 5`)
	if err != nil {
		internalErr(w, err)
		return
	}
	defer qrows.Close()
	for qrows.Next() {
		var id, name string
		var score int
		var at time.Time
		if err := qrows.Scan(&id, &name, &score, &at); err != nil {
			continue
		}
		items = append(items, notificationItem{
			ID: "high_value_qualified:" + id, Type: "high_value_qualified",
			Title: "سرنخ پرارزش واجد شرایط شد", Body: name + " (امتیاز " + strconv.Itoa(score) + ")",
			Link: "/leads/" + id, CreatedAt: at, Read: markRead(at),
		})
	}

	// Sort by recency, most recent first.
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].CreatedAt.After(items[j-1].CreatedAt); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	if len(items) > 30 {
		items = items[:30]
	}
	RespondOK(w, items)
}

// MarkNotificationsRead stores a per-user "read" marker.
func (h *systemHandler) MarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	if h.cache != nil {
		_ = h.cache.Set(r.Context(), "notif:read:"+p.UserID, time.Now().UTC().Format(time.RFC3339), 30*24*time.Hour)
	}
	RespondNoContent(w)
}
