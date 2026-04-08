package adminapi

import (
	"errors"
	"net/http"
	"time"
)

// PublicError is an error type safe to expose to dashboard users.
// It carries an HTTP status code and an optional retry delay.
type PublicError struct {
	Status     int
	Message    string
	RetryAfter time.Duration
}

func (e *PublicError) Error() string {
	return e.Message
}

func (e *PublicError) statusCode() int {
	if e == nil || e.Status == 0 {
		return http.StatusBadRequest
	}
	return e.Status
}

func asPublicError(err error) (*PublicError, bool) {
	var pe *PublicError
	if errors.As(err, &pe) && pe != nil {
		return pe, true
	}
	return nil, false
}
