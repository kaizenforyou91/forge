package agent

import "errors"

var (
	ErrInvalidRun  = errors.New("agent: invalid run")
	ErrRunConsumed = errors.New("agent: run already consumed")
)
