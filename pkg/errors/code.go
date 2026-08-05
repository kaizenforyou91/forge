package errors

// Code represents an application error code.
type Code string

const (
	CodeUnknown         Code = "UNKNOWN"
	CodeInvalidConfig   Code = "INVALID_CONFIG"
	CodeInvalidManifest Code = "INVALID_MANIFEST"
	CodeNotFound        Code = "NOT_FOUND"
	CodeInternal        Code = "INTERNAL"
)
