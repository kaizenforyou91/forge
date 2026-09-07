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
// and causes. Joined failures remain joined (e.g. cancellation plus cleanup).
func SafeError(err error) error {
	if err == nil {
		return nil
	}
	var categories []error
	for _, category := range []error{
		context.Canceled, context.DeadlineExceeded,
		ErrInvalidRequest, ErrAuthentication, ErrAuthorization, ErrRateLimited,
		ErrQuotaExceeded, ErrTransport, ErrMalformedResponse, ErrResponseTooLarge,
		ErrIncompleteResponse, ErrRefused, ErrProvider,
	} {
		if errors.Is(err, category) {
			categories = append(categories, category)
		}
	}
	if len(categories) == 0 {
		return ErrProvider
	}
	if len(categories) == 1 {
		return categories[0]
	}
	return errors.Join(categories...)
}
