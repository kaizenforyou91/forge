package plugin

import (
	"errors"
	"sync"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/config"
)

var ErrDuplicatePlugin = errors.New("plugin already registered")

// Registry stores registered Forge plugins.
//
// Registry is responsible only for plugin identity and lookup.
// Plugin lifecycle remains owned by app.Runtime/app.App.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	order   []string
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
		order:   make([]string, 0),
	}
}

// Register adds a plugin to the registry.
//
// Plugin names must be unique within a registry.
func (r *Registry) Register(plugin Plugin) error {
	if plugin == nil {
		return errors.New("plugin is nil")
	}

	name := plugin.Name()
	if name == "" {
		return errors.New("plugin name is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		return ErrDuplicatePlugin
	}

	r.plugins[name] = plugin
	r.order = append(r.order, name)

	return nil
}

// Get returns a registered plugin by name.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.plugins[name]
	return plugin, ok
}

// List returns registered plugins in registration order.
func (r *Registry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Plugin, 0, len(r.order))

	for _, name := range r.order {
		result = append(result, r.plugins[name])
	}

	return result
}

// Use registers all plugins in this registry into the application.
//
// Plugins are registered in deterministic registration order.
// Plugin lifecycle methods are not invoked here; lifecycle remains owned
// by app.App.
func (r *Registry) Use(a *app.App) error {
	for _, p := range r.List() {
		if err := a.Add(p); err != nil {
			return err
		}
	}

	return nil
}

// UseConfigured registers enabled plugins into the application.
//
// Plugins remain in deterministic registry order. Disabled plugins are
// skipped. Plugin lifecycle remains owned by app.App.
func (r *Registry) UseConfigured(a *app.App, cfg config.Config) error {
	for _, p := range r.List() {
		if !cfg.Plugins.IsEnabled(p.Name()) {
			continue
		}

		if err := a.Add(p); err != nil {
			return err
		}
	}

	return nil
}
