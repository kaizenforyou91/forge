package compiler

import "errors"

var (
	ErrInvalidBuildPlan        = errors.New("invalid build plan")
	ErrNilCompiler             = errors.New("compiler is nil")
	ErrNilExecutor             = errors.New("executor is nil")
	ErrNilCommandRunner        = errors.New("command runner is nil")
	ErrCommandFailed           = errors.New("command execution failed")
	ErrNilArtifactWriter       = errors.New("artifact writer is nil")
	ErrInvalidArtifact         = errors.New("invalid artifact")
	ErrInvalidOutputRoot       = errors.New("invalid artifact output root")
	ErrInvalidArtifactBundle   = errors.New("invalid artifact bundle")
	ErrInvalidArtifactPackage  = errors.New("invalid artifact package")
	ErrMissingArtifactPayload  = errors.New("missing artifact payload")
	ErrInvalidPackageIntegrity = errors.New("invalid package integrity")
	ErrIntegrityMismatch       = errors.New("package integrity mismatch")
	ErrMissingPackageIntegrity = errors.New("missing package integrity")
)
