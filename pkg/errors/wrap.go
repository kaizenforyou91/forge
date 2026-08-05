package errors

// New creates a new Forge error.
func New(code Code, message string) error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error.
func Wrap(err error, code Code, message string) error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
