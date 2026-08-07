package container

import "errors"

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrInvalidTarget   = errors.New("target must be a non-nil pointer")
)
var ErrInvalidConstructor = errors.New("invalid constructor")

var ErrContainerFrozen = errors.New("container is frozen")
