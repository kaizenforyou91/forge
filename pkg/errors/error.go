package errors

import "fmt"

// Error represents a Forge application error.
type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}

	return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
}

// Unwrap enables errors.Unwrap().
func (e *Error) Unwrap() error {
	return e.Err
}
