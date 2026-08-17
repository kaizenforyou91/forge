package cli

import (
	"context"
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/app"
)

type applicationContextKey struct{}

// NewApplicationContext stores the Forge application in a context.
func NewApplicationContext(application *app.App) context.Context {
	if application == nil {
		return context.Background()
	}

	return context.WithValue(
		context.Background(),
		applicationContextKey{},
		application,
	)
}

// ApplicationFromContext returns the Forge application stored in a command context.
func ApplicationFromContext(ctx context.Context) (*app.App, error) {
	if ctx == nil {
		return nil, fmt.Errorf("application context is nil")
	}

	application, ok := ctx.Value(applicationContextKey{}).(*app.App)
	if !ok || application == nil {
		return nil, fmt.Errorf("application is not configured")
	}

	return application, nil
}
