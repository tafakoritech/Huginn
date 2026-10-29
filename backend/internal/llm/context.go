package llm

import (
	"context"
	"time"
)

// withoutDeadlineCtx forwards the parent's cancellation and values but drops
// its wall-clock deadline. This lets each individual HTTP attempt receive a
// fresh, full request timeout instead of inheriting a nearly-expired deadline
// from the surrounding job context (which may have been partially consumed by
// earlier retries, formats, or chunks).
type withoutDeadlineCtx struct {
	context.Context
}

// Deadline always reports "no deadline".
func (withoutDeadlineCtx) Deadline() (time.Time, bool) { return time.Time{}, false }

// WithoutDeadline returns a context that keeps the parent's values and
// cancellation but drops its wall-clock deadline. Use it to give a bounded
// sub-operation its own full timeout budget even when an enclosing context has
// a much shorter deadline (e.g. a generic 30s HTTP request deadline around a
// call-scenario generation that legitimately takes minutes). Parent
// cancellation — a client disconnecting, the user leaving the screen, server
// shutdown — still propagates.
func WithoutDeadline(parent context.Context) context.Context {
	return withoutDeadlineCtx{Context: parent}
}

// attemptContext returns a fresh context for a single network attempt with the
// full configured request timeout. Cancellation of the parent (the overall
// document/job context) still propagates, but the parent's deadline no longer
// shrinks the per-attempt budget.
//
// This satisfies: "Each network attempt must receive its own fresh
// context.WithTimeout(...). Keep cancellation tied to the overall document/job
// context, but separate overall-job lifetime from individual HTTP-attempt
// timeout. Ensure retry attempt N receives the full configured request timeout
// unless the overall job itself is explicitly cancelled."
func attemptContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(withoutDeadlineCtx{Context: parent}, timeout)
}
