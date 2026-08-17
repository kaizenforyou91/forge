package bootstrap

import (
	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/compiler"
)

// NewApplication creates the Forge application with the core modules.
func NewApplication() *app.App {
	return app.New().
		Use(compiler.NewModule())
}
