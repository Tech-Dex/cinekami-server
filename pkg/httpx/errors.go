package httpx

import (
	"errors"
	"fmt"
	"net/http"
)

// HTTPError standardizes API error responses and logging context.
type HTTPError struct {
	StatusCode int            `json:"-"`
	Message    string         `json:"message"`
	Code       string         `json:"code"`
	Details    map[string]any `json:"details,omitempty"`
	Err        error          `json:"-"`
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *HTTPError) Unwrap() error { return e.Err }

// Helpers
func BadRequest(msg string, err error) *HTTPError {
	return &HTTPError{StatusCode: http.StatusBadRequest, Message: msg, Code: "bad_request", Err: err}
}
func Unauthorized(msg string, err error) *HTTPError {
	return &HTTPError{StatusCode: http.StatusUnauthorized, Message: msg, Code: "unauthorized", Err: err}
}
func Forbidden(msg string, err error) *HTTPError {
	return &HTTPError{StatusCode: http.StatusForbidden, Message: msg, Code: "forbidden", Err: err}
}
func NotFound(msg string, err error) *HTTPError {
	return &HTTPError{StatusCode: http.StatusNotFound, Message: msg, Code: "not_found", Err: err}
}
func Conflict(msg string, err error) *HTTPError {
	return &HTTPError{StatusCode: http.StatusConflict, Message: msg, Code: "conflict", Err: err}
}
func Internal(msg string, err error) *HTTPError {
	return &HTTPError{StatusCode: http.StatusInternalServerError, Message: msg, Code: "internal", Err: err}
}

// Is compares target code regardless of wrapped error.
func Is(err error, code string) bool {
	var he *HTTPError
	if errors.As(err, &he) {
		return he.Code == code
	}
	return false
}
