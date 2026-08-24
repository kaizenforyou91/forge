package runtime

import "errors"

var (
	ErrPackageNotRunnable         = errors.New("package is not runnable")
	ErrUnsupportedRuntimePlatform = errors.New("unsupported runtime platform")
	ErrInvalidRunnablePackage     = errors.New("invalid runnable package")
)
