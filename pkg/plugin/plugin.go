package plugin

import "github.com/kaizenforyou91/forge/pkg/app"

// Plugin represents a Forge plugin.
//
// A plugin is an application module with version metadata.
// Plugin lifecycle is delegated to app.Module so Forge keeps
// a single lifecycle implementation in pkg/app.
type Plugin interface {
	app.Module
	Version() string
}
