package app

import "context"

// Runtime represents the application runtime.
type Runtime struct {
	context context.Context

	cancel context.CancelFunc
}
