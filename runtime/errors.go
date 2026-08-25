package runtime

import "errors"

var (
	ErrPackageNotRunnable                = errors.New("package is not runnable")
	ErrUnsupportedRuntimePlatform        = errors.New("unsupported runtime platform")
	ErrInvalidRunnablePackage            = errors.New("invalid runnable package")
	ErrExecutableMaterializationFailed   = errors.New("executable materialization failed")
	ErrMaterializedExecutableInvalid     = errors.New("materialized executable is invalid")
	ErrMaterializedExecutableClosed      = errors.New("materialized executable is closed")
	ErrMaterializedExecutableBusy        = errors.New("materialized executable is busy")
	ErrMaterializedExecutableAlreadyUsed = errors.New("materialized executable was already used")
)
