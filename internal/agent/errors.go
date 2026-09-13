package agent

import "errors"

var (
	ErrInvalidRun     = errors.New("agent: invalid run")
	ErrRunConsumed    = errors.New("agent: run already consumed")
	ErrInvalidHost    = errors.New("agent: invalid host")
	ErrHostNotRunning = errors.New("agent: host not running")
)
