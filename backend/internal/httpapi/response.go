package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// APIError is a machine-readable error response.
type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// Error codes used across the API.
const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeValidationError     = "VALIDATION_ERROR"
	CodeInternalError       = "INTERNAL_ERROR"
	CodeMandatoryReport     = "MANDATORY_REPORT_PENDING"
	CodeInvalidTransition   = "INVALID_STATE_TRANSITION"
	CodeDuplicateExport     = "DUPLICATE_EXPORT"
	CodeRateLimited         = "RATE_LIMITED"
	CodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	CodePayloadTooLarge     = "PAYLOAD_TOO_LARGE"
	CodeUnsupportedMedia    = "UNSUPPORTED_MEDIA_TYPE"
)

// RespondError writes a consistent error envelope.
func RespondError(w http.ResponseWriter, status int, e *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": e})
}

// RespondOK writes a JSON body with 200.
func RespondOK(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

// RespondJSON writes arbitrary JSON with a given status.
func RespondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// RespondNoContent writes 204.
func RespondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// DecodeJSON parses a request body with size limits.
func DecodeJSON(r *http.Request, out any) *APIError {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return &APIError{Code: CodeBadRequest, Message: "invalid JSON body: " + err.Error()}
	}
	return nil
}

// ErrorFrom maps sentinel errors to API errors.
func ErrorFrom(err error) (int, *APIError) {
	if err == nil {
		return http.StatusOK, nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		status := statusForCode(apiErr.Code)
		return status, apiErr
	}
	return http.StatusInternalServerError, &APIError{
		Code:    CodeInternalError,
		Message: "internal server error",
	}
}

func statusForCode(code string) int {
	switch code {
	case CodeBadRequest, CodeValidationError, CodeInvalidTransition, CodePayloadTooLarge, CodeUnsupportedMedia:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden, CodeMandatoryReport, CodeDuplicateExport:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// ParseTime parses RFC3339 timestamps.
func ParseTime(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }
