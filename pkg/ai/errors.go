package ai

import (
	"context"
	"errors"
)

// Errors classify failures without exposing request, credential or response data.
var (
	ErrInvalidRequest     = errors.New("ai: invalid request")
	ErrAuthentication     = errors.New("ai: authentication failed")
	ErrAuthorization      = errors.New("ai: authorization denied")
	ErrRateLimited        = errors.New("ai: rate limited")
	ErrQuotaExceeded      = errors.New("ai: quota exceeded")
	ErrTransport          = errors.New("ai: transport failed")
	ErrMalformedResponse  = errors.New("ai: malformed response")
	ErrResponseTooLarge   = errors.New("ai: response too large")
	ErrIncompleteResponse = errors.New("ai: incomplete response")
	ErrRefused            = errors.New("ai: response refused")
	ErrProvider           = errors.New("ai: provider failed")
)

// SafeError retains known classifications but discards untrusted error messages
// and causes. Every unknown failure branch contributes ErrProvider, so a mixed
// failure cannot become pure cancellation. Categories have a stable order.
func SafeError(err error) error {
	if err == nil {
		return nil
	}
	known := []error{
		context.Canceled, context.DeadlineExceeded,
		ErrInvalidRequest, ErrAuthentication, ErrAuthorization, ErrRateLimited,
		ErrQuotaExceeded, ErrTransport, ErrMalformedResponse, ErrResponseTooLarge,
		ErrIncompleteResponse, ErrRefused, ErrProvider,
	}
	found := make(map[error]bool)
	collectSafeCategories(err, known, found)
	var categories []error
	for _, category := range known {
		if found[category] {
			categories = append(categories, category)
		}
	}
	if len(categories) == 1 {
		return categories[0]
	}
	return errors.Join(categories...)
}

func collectSafeCategories(err error, known []error, found map[error]bool) {
	if err == nil {
		return
	}
	// Match this node only. A match elsewhere in the tree must not hide an
	// unknown sibling. Retain custom Is classifications without retaining err.
	matcher, _ := err.(interface{ Is(error) bool })
	matched := false
	for _, category := range known {
		if err == category || matcher != nil && matcher.Is(category) {
			found[category] = true
			matched = true
		}
	}
	var children []error
	switch wrapped := err.(type) {
	case interface{ Unwrap() []error }:
		children = wrapped.Unwrap()
	case interface{ Unwrap() error }:
		children = []error{wrapped.Unwrap()}
	}
	hasChild := false
	for _, child := range children {
		if child != nil {
			hasChild = true
			collectSafeCategories(child, known, found)
		}
	}
	// A wrapper with a cause is context, not an extra failure. An unclassified
	// leaf (including a wrapper without a cause) represents an unknown failure.
	if !hasChild && !matched {
		found[ErrProvider] = true
	}
}
