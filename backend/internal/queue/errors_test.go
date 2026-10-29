package queue

import (
	"errors"
	"fmt"
	"testing"
)

type fakeHTTPError struct {
	code int
}

func (f *fakeHTTPError) Error() string            { return fmt.Sprintf("upstream %d", f.code) }
func (f *fakeHTTPError) HTTPStatusCode() int      { return f.code }

func TestIsPermanent(t *testing.T) {
	permanent := NewPermanent(CodeDocumentNotFound, errors.New("gone"))
	if !IsPermanent(permanent) {
		t.Fatal("expected IsPermanent true")
	}
	wrapped := fmt.Errorf("wrap: %w", permanent)
	if !IsPermanent(wrapped) {
		t.Fatal("expected IsPermanent true through wrapping")
	}
	if IsPermanent(errors.New("generic")) {
		t.Fatal("generic errors must not be permanent")
	}
	if IsPermanent(nil) {
		t.Fatal("nil is not permanent")
	}
}

func TestPermanentCode(t *testing.T) {
	if got := PermanentCode(NewPermanent(CodeDocumentNotFound, nil)); got != CodeDocumentNotFound {
		t.Fatalf("PermanentCode = %s, want DOCUMENT_NOT_FOUND", got)
	}
	if got := PermanentCode(errors.New("generic")); got != "" {
		t.Fatalf("PermanentCode for generic = %q, want empty", got)
	}
}

func TestIsRetryableDefaultsTransient(t *testing.T) {
	// Unclassified errors (DB down, network reset, MinIO timeout) are retryable.
	for _, err := range []error{
		errors.New("postgres connection refused"),
		errors.New("connection reset by peer"),
		fmt.Errorf("minio timeout: %w", errors.New("context deadline exceeded")),
	} {
		if !IsRetryable(err) {
			t.Fatalf("expected %q to be retryable", err)
		}
	}
}

func TestIsRetryablePermanent(t *testing.T) {
	if IsRetryable(NewPermanent(CodeDocumentNotFound, errors.New("x"))) {
		t.Fatal("permanent errors must not be retryable")
	}
}

func TestHTTPStatusClassification(t *testing.T) {
	retryable := []int{408, 429, 500, 502, 503, 504}
	permanent := []int{400, 401, 403, 404, 422}

	for _, code := range retryable {
		if !IsRetryable(&fakeHTTPError{code: code}) {
			t.Errorf("HTTP %d should be retryable", code)
		}
	}
	for _, code := range permanent {
		if IsRetryable(&fakeHTTPError{code: code}) {
			t.Errorf("HTTP %d should NOT be retryable", code)
		}
	}
	// transport-level (status 0) is transient
	if !IsRetryable(&fakeHTTPError{code: 0}) {
		t.Error("HTTP 0 (transport) should be retryable")
	}
	// classification works through wrapping
	if IsRetryable(fmt.Errorf("wrapped: %w", &fakeHTTPError{code: 401})) {
		t.Error("401 should stay non-retryable through wrapping")
	}
}

func TestPermanentErrorUnwrap(t *testing.T) {
	cause := errors.New("cause")
	permanent := NewPermanent(CodeInvalidUUID, cause)
	if !errors.Is(permanent, cause) {
		t.Fatal("errors.Is should see the underlying cause")
	}
}
