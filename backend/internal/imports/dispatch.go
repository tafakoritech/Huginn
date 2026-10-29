package imports

import (
	"context"
	"log/slog"
	"time"

	"github.com/tafakorpardazan/huginn/internal/queue"
)

// OutboxDocumentExtract is the event type written atomically with a document.
const OutboxDocumentExtract = "import.document.extract"

// PublishFunc publishes a queue job.
type PublishFunc func(ctx context.Context, job queue.Job) error

const maxOutboxAttempts = 10

// DispatchOutbox publishes pending outbox processing triggers to the queue.
// It is idempotent: events are claimed atomically (PENDING -> PROCESSING), so
// concurrent dispatchers never publish the same event twice.
func DispatchOutbox(ctx context.Context, svc *Service, publish PublishFunc) error {
	events, err := svc.ListPendingOutbox(ctx, 100)
	if err != nil {
		return err
	}
	for _, e := range events {
		if e.EventType != OutboxDocumentExtract {
			continue
		}
		docID, _ := e.Payload["document_id"].(string)
		batchID, _ := e.Payload["batch_id"].(string)
		if docID == "" || batchID == "" {
			_ = svc.DeadOutbox(ctx, e.ID)
			continue
		}
		claimed, err := svc.ClaimOutbox(ctx, e.ID)
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}
		job := queue.Job{
			Type:      queue.JobDocumentExtract,
			ID:        docID,
			Payload:   map[string]any{"document_id": docID, "batch_id": batchID},
			CreatedAt: time.Now(),
		}
		if err := publish(ctx, job); err != nil {
			next := time.Now().Add(backoff(e.Attempts))
			_ = svc.FailOutbox(ctx, e.ID, err.Error(), &next)
			continue
		}
		_ = svc.FinishOutbox(ctx, e.ID)
	}
	return nil
}

func backoff(attempts int) time.Duration {
	base := 2 * time.Second
	d := time.Duration(attempts) * base
	if d > 5*time.Minute {
		return 5 * time.Minute
	}
	return d
}

// RunOutboxDispatcher polls pending outbox events and publishes them until the
// context is cancelled. Used by the API and worker processes for reliability.
func RunOutboxDispatcher(ctx context.Context, logger *slog.Logger, svc *Service, publish PublishFunc, interval time.Duration) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			if err := DispatchOutbox(ctx, svc, publish); err != nil {
				logger.Warn("outbox dispatch failed", "error", err)
			}
		}
	}
}
