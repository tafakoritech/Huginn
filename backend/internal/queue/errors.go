package queue

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable, machine-readable category for a job failure. It is
// used in structured worker logs and DLQ context so operators can triage
// dead-lettered jobs without parsing human-readable strings.
type ErrorCode string

// Failure categories for queue processing.
const (
	CodeDocumentNotFound   ErrorCode = "DOCUMENT_NOT_FOUND"
	CodeBatchNotFound      ErrorCode = "BATCH_NOT_FOUND"
	CodeInvalidPayload     ErrorCode = "INVALID_PAYLOAD"
	CodeInvalidUUID        ErrorCode = "INVALID_UUID"
	CodeInvalidState       ErrorCode = "INVALID_DOCUMENT_STATE"
	CodeValidation         ErrorCode = "VALIDATION_ERROR"
	CodeProviderRejected   ErrorCode = "PROVIDER_REJECTED"
	CodeNotFound           ErrorCode = "NOT_FOUND"
	CodePermanentGeneric   ErrorCode = "PERMANENT"
)

// PermanentError is a non-retryable job failure. Messages that fail with a
// PermanentError are never requeued; they are dead-lettered instead.
//
// A permanent failure can never succeed by retrying the same input, for
// example when the referenced document no longer exists or the job payload is
// malformed.
type PermanentError struct {
	Code ErrorCode
	Err  error
}

func (e *PermanentError) Error() string {
	if e.Err == nil {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %v", e.Code, e.Err)
}

// Unwrap exposes the underlying error for errors.Is/errors.As.
func (e *PermanentError) Unwrap() error { return e.Err }

// NewPermanent wraps err as a permanent, non-retryable failure.
func NewPermanent(code ErrorCode, err error) error {
	if err == nil {
		err = errors.New(string(code))
	}
	return &PermanentError{Code: code, Err: err}
}

// NewPermanentf builds a permanent failure from a format string.
func NewPermanentf(code ErrorCode, format string, args ...any) error {
	return &PermanentError{Code: code, Err: fmt.Errorf(format, args...)}
}

// IsPermanent reports whether err is (or wraps) a PermanentError.
func IsPermanent(err error) bool {
	if err == nil {
		return false
	}
	var pe *PermanentError
	return errors.As(err, &pe)
}

// HTTPStatusCarrier is implemented by errors that carry an upstream HTTP
// status code (e.g. the LLM provider). Queue classification uses it to decide
// whether an upstream HTTP failure is transient or permanent.
type HTTPStatusCarrier interface {
	HTTPStatusCode() int
}

// Retryable HTTP statuses: the request could succeed if retried later.
var retryableHTTPStatuses = map[int]bool{
	httpStatusRequestTimeout:      true, // 408
	httpStatusTooManyRequests:     true, // 429
	httpStatusInternal:            true, // 500
	httpStatusBadGateway:          true, // 502
	httpStatusServiceUnavailable:  true, // 503
	httpStatusGatewayTimeout:      true, // 504
}

// Permanent 4xx client errors: retrying the identical request cannot succeed.
var permanentHTTPStatuses = map[int]bool{
	httpStatusBadRequest:   true, // 400
	httpStatusUnauthorized: true, // 401
	httpStatusForbidden:    true, // 403
	httpStatusNotFound:     true, // 404
	httpStatusUnprocessable: true, // 422
}

// IsRetryable reports whether a job failure may be retried. Permanent errors
// and definitively-rejected HTTP responses are non-retryable; everything else
// (transport errors, 408/429/5xx, generic transient failures) is retryable.
func IsRetryable(err error) bool {
	if err == nil {
		return true
	}
	if IsPermanent(err) {
		return false
	}
	var hc HTTPStatusCarrier
	if errors.As(err, &hc) {
		code := hc.HTTPStatusCode()
		if permanentHTTPStatuses[code] {
			return false
		}
		if retryableHTTPStatuses[code] {
			return true
		}
	}
	// Unknown/transport errors default to retryable so a transient outage
	// (PostgreSQL down, MinIO timeout, network reset, Tika hiccup) recovers.
	return true
}

// PermanentCode returns the classification code for err, empty when retryable.
func PermanentCode(err error) ErrorCode {
	if !IsPermanent(err) {
		return ""
	}
	var pe *PermanentError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return CodePermanentGeneric
}

// plain numeric HTTP codes (avoid importing net/http in the errors package
// to keep it dependency-light for consumers).
const (
	httpStatusBadRequest       = 400
	httpStatusUnauthorized     = 401
	httpStatusForbidden        = 403
	httpStatusNotFound         = 404
	httpStatusRequestTimeout   = 408
	httpStatusUnprocessable    = 422
	httpStatusTooManyRequests  = 429
	httpStatusInternal         = 500
	httpStatusBadGateway       = 502
	httpStatusServiceUnavailable = 503
	httpStatusGatewayTimeout   = 504
)
