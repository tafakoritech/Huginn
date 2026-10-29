package llm

import (
	"context"
	"testing"
	"time"
)

// TestAttemptContextDropsParentDeadline verifies the core timeout guarantee:
// a per-attempt context gets its OWN full timeout budget even when the parent
// context has a much shorter deadline. This is what keeps call-scenario
// generation (LLM timeout 180s) alive when an enclosing HTTP request context
// would otherwise expire after 30s.
func TestAttemptContextDropsParentDeadline(t *testing.T) {
	parent, cancelParent := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelParent()

	attempt, cancel := attemptContext(parent, 5*time.Second)
	defer cancel()

	deadline, ok := attempt.Deadline()
	if !ok {
		t.Fatal("attempt context must have a deadline")
	}
	remaining := time.Until(deadline)
	if remaining < 4*time.Second {
		t.Fatalf("attempt context must keep its full timeout, remaining=%v", remaining)
	}
}

// TestAttemptContextPropagatesCancellation verifies that intentionally cancelling
// the parent (client disconnect, user leaving the screen) DOES propagate to the
// per-attempt context — cancellation is preserved even though the deadline is
// dropped.
func TestAttemptContextPropagatesCancellation(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	attempt, cancel := attemptContext(parent, 5*time.Second)
	defer cancel()

	cancelParent()
	select {
	case <-attempt.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("parent cancellation must propagate to the attempt context")
	}
}

// TestAttemptContextKeepsValues verifies the attempt context forwards context
// values from the parent.
func TestAttemptContextKeepsValues(t *testing.T) {
	parent := context.WithValue(context.Background(), "k", "v")
	attempt, cancel := attemptContext(parent, 5*time.Second)
	defer cancel()
	if attempt.Value("k") != "v" {
		t.Fatal("attempt context must forward parent values")
	}
}
