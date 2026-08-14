package compiler

import "errors"

var (
	ErrInvalidBuildPlan = errors.New("invalid build plan")
	ErrNilCompiler      = errors.New("compiler is nil")
	ErrNilExecutor      = errors.New("executor is nil")
)
