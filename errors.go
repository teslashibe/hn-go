package hn

import (
	"errors"
	"fmt"
)

// Sentinel errors.
var (
	ErrUnauthenticated = errors.New("hn: missing or invalid user cookie")
	ErrNotFound        = errors.New("hn: item not found")
	ErrForbidden       = errors.New("hn: action forbidden (insufficient karma or invalid auth token)")
	ErrRateLimited     = errors.New("hn: rate limited")
	ErrParseFailure    = errors.New("hn: failed to parse page")
)

// HTTPError is returned for unexpected non-2xx / non-3xx HTTP responses.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("hn: HTTP %d: %s", e.StatusCode, e.Body)
}
