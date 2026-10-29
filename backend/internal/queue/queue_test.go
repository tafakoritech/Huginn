package queue

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ---- fakes ----

type fakeAck struct {
	mu     sync.Mutex
	acks   []uint64
	nacks  []nackRecord
}

type nackRecord struct {
	tag      uint64
	requeue  bool
}

func (f *fakeAck) Ack(tag uint64, multiple bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.acks = append(f.acks, tag)
	return nil
}

func (f *fakeAck) Nack(tag uint64, multiple bool, requeue bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nacks = append(f.nacks, nackRecord{tag: tag, requeue: requeue})
	return nil
}

func (f *fakeAck) Reject(tag uint64, requeue bool) error { return f.Nack(tag, false, requeue) }

func (f *fakeAck) ackCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.acks)
}

func (f *fakeAck) nackCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.nacks)
}

func (f *fakeAck) lastNack() (nackRecord, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.nacks) == 0 {
		return nackRecord{}, false
	}
	return f.nacks[len(f.nacks)-1], true
}

type fakePublisher struct {
	mu        sync.Mutex
	published []amqp.Publishing
}

func (f *fakePublisher) PublishWithContext(_ context.Context, _, _ string, _, _ bool, msg amqp.Publishing) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published = append(f.published, msg)
	return nil
}

func (f *fakePublisher) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.published)
}

func (f *fakePublisher) body(i int) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.published[i].Body
}

func newTestClient() (*Client, *fakeAck, *fakePublisher) {
	return &Client{
			logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
			retryLimit:     2,
			retryBaseDelay: time.Millisecond,
		},
		&fakeAck{},
		&fakePublisher{}
}

func delivery(body []byte, ack *fakeAck, tag uint64) amqp.Delivery {
	return amqp.Delivery{Acknowledger: ack, DeliveryTag: tag, Body: body}
}

func jobBody(t *testing.T, job Job) []byte {
	t.Helper()
	b, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decodeJob(t *testing.T, body []byte) Job {
	t.Helper()
	var j Job
	if err := json.Unmarshal(body, &j); err != nil {
		t.Fatal(err)
	}
	return j
}

// ---- tests ----

func TestHandleDeliverySuccessAcks(t *testing.T) {
	c, _, _ := newTestClient()
	ok := &fakeAck{}
	d := delivery(jobBody(t, Job{Type: JobDocumentExtract, ID: "1"}), ok, 1)

	c.handleDelivery(context.Background(), &fakePublisher{}, d, func(ctx context.Context, j Job) error {
		return nil
	})

	if ok.ackCount() != 1 || ok.nackCount() != 0 {
		t.Fatalf("success must ACK once, got acks=%d nacks=%d", ok.ackCount(), ok.nackCount())
	}
}

// Case A (queue layer): a permanent error must dead-letter immediately with a
// single processing attempt and no requeue.
func TestHandleDeliveryPermanentDeadLettersImmediately(t *testing.T) {
	c, ack, pub := newTestClient()
	attempts := 0
	d := delivery(jobBody(t, Job{Type: JobDocumentExtract, ID: "stale"}), ack, 1)

	c.handleDelivery(context.Background(), pub, d, func(ctx context.Context, j Job) error {
		attempts++
		return NewPermanent(CodeDocumentNotFound, errors.New("not found"))
	})

	if attempts != 1 {
		t.Fatalf("permanent job must be attempted exactly once, got %d", attempts)
	}
	if ack.nackCount() != 1 {
		t.Fatalf("permanent job must NACK once, got %d", ack.nackCount())
	}
	if last, ok := ack.lastNack(); !ok || last.requeue {
		t.Fatalf("permanent job must NACK with requeue=false (DLQ), got %+v", last)
	}
	if pub.count() != 0 {
		t.Fatalf("permanent job must NOT be republished, got %d publishes", pub.count())
	}
	if ack.ackCount() != 0 {
		t.Fatal("permanent job must not be acked")
	}
}

// Case B (queue layer): a transient failure schedules a delayed retry.
func TestHandleDeliveryTransientRepublishes(t *testing.T) {
	c, ack, pub := newTestClient()
	d := delivery(jobBody(t, Job{Type: JobDocumentExtract, ID: "x", Retry: 0}), ack, 1)

	c.handleDelivery(context.Background(), pub, d, func(ctx context.Context, j Job) error {
		return errors.New("postgres temporarily unavailable")
	})

	time.Sleep(20 * time.Millisecond) // allow the delayed republish goroutine

	if pub.count() != 1 {
		t.Fatalf("transient failure should be republished for retry, got %d", pub.count())
	}
	republished := decodeJob(t, pub.body(0))
	if republished.Retry != 1 {
		t.Fatalf("republished job should carry Retry=1, got %d", republished.Retry)
	}
	if ack.ackCount() != 1 {
		t.Fatal("original delivery should be ACKed after successful republish")
	}
	if ack.nackCount() != 0 {
		t.Fatal("transient failure must not dead-letter on first attempt")
	}
}

// Case C: a transient failure that recovers eventually ACKs and is never DLQ'd.
func TestHandleDeliveryTransientRecovers(t *testing.T) {
	c, ack, pub := newTestClient()
	failFirst := true
	d := delivery(jobBody(t, Job{Type: JobDocumentExtract, ID: "x", Retry: 0}), ack, 1)

	handler := func(ctx context.Context, j Job) error {
		if failFirst {
			failFirst = false
			return errors.New("connection reset")
		}
		return nil
	}

	// attempt 1: transient -> republish
	c.handleDelivery(context.Background(), pub, d, handler)
	time.Sleep(20 * time.Millisecond)
	if pub.count() != 1 {
		t.Fatalf("expected 1 republish, got %d", pub.count())
	}

	// attempt 2 (the republished message): now succeeds -> ACK
	ack2 := &fakeAck{}
	d2 := delivery(pub.body(0), ack2, 2)
	c.handleDelivery(context.Background(), pub, d2, handler)

	if ack2.ackCount() != 1 || ack2.nackCount() != 0 {
		t.Fatalf("recovered job must ACK and never DLQ: acks=%d nacks=%d", ack2.ackCount(), ack2.nackCount())
	}
	if pub.count() != 1 {
		t.Fatalf("recovered job must not republish again, got %d", pub.count())
	}
}

// Case D: a persistent transient failure dead-letters after the retry limit.
func TestHandleDeliveryRetryLimitExhausted(t *testing.T) {
	c, _, _ := newTestClient() // retryLimit=2
	pub := &fakePublisher{}
	handler := func(ctx context.Context, j Job) error { return errors.New("minio timeout") }

	// attempt 1 (Retry=0) -> republish Retry=1
	ack1 := &fakeAck{}
	d1 := delivery(jobBody(t, Job{Type: JobDocumentExtract, ID: "y", Retry: 0}), ack1, 1)
	c.handleDelivery(context.Background(), pub, d1, handler)
	time.Sleep(20 * time.Millisecond)
	if pub.count() != 1 {
		t.Fatalf("expected 1 publish after attempt 1, got %d", pub.count())
	}

	// attempt 2 (Retry=1) -> republish Retry=2
	ack2 := &fakeAck{}
	d2 := delivery(pub.body(0), ack2, 2)
	c.handleDelivery(context.Background(), pub, d2, handler)
	time.Sleep(20 * time.Millisecond)
	if pub.count() != 2 {
		t.Fatalf("expected 2 publishes after attempt 2, got %d", pub.count())
	}

	// attempt 3 (Retry=2) -> attempt 3 > limit 2 -> DLQ
	ack3 := &fakeAck{}
	d3 := delivery(pub.body(1), ack3, 3)
	c.handleDelivery(context.Background(), pub, d3, handler)
	time.Sleep(20 * time.Millisecond)

	if ack3.nackCount() != 1 {
		t.Fatalf("exhausted retries must NACK once, got %d", ack3.nackCount())
	}
	if last, ok := ack3.lastNack(); !ok || last.requeue {
		t.Fatalf("exhausted retries must NACK with requeue=false (DLQ), got %+v", last)
	}
	if pub.count() != 2 {
		t.Fatalf("no further republish after exhaustion, got %d", pub.count())
	}
}

func TestHandleDeliveryMalformedBodyDeadLetters(t *testing.T) {
	c, ack, pub := newTestClient()
	handlerCalled := false
	d := delivery([]byte("not json"), ack, 1)

	c.handleDelivery(context.Background(), pub, d, func(ctx context.Context, j Job) error {
		handlerCalled = true
		return nil
	})

	if handlerCalled {
		t.Fatal("handler must not run for malformed body")
	}
	if last, ok := ack.lastNack(); !ok || last.requeue {
		t.Fatal("malformed body must NACK with requeue=false (DLQ)")
	}
	if pub.count() != 0 {
		t.Fatal("malformed body must not be republished")
	}
}

func TestRetryDelayExponential(t *testing.T) {
	c := &Client{retryBaseDelay: time.Second, retryLimit: 5}
	// 1-based attempt: 1s, 2s, 4s, 8s, 16s
	want := []time.Duration{1, 2, 4, 8, 16}
	for i, w := range want {
		got := c.retryDelay(i + 1)
		if got != w*time.Second {
			t.Errorf("attempt %d delay = %v, want %v", i+1, got, w*time.Second)
		}
	}
	// capped at 1 minute
	if got := c.retryDelay(10); got != time.Minute {
		t.Errorf("delay must cap at 1m, got %v", got)
	}
}
