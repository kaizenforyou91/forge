package plugin

import (
	"errors"
	"sync"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/app"
)

type registryTestPlugin struct {
	name    string
	version string
}

func (p *registryTestPlugin) Name() string {
	return p.name
}

func (p *registryTestPlugin) Version() string {
	return p.version
}

func (p *registryTestPlugin) Register(*app.App) error {
	return nil
}

func (p *registryTestPlugin) Start(*app.App) error {
	return nil
}

func (p *registryTestPlugin) Stop(*app.App) error {
	return nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	p := &registryTestPlugin{
		name:    "logger",
		version: "1.0.0",
	}

	if err := r.Register(p); err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}

	got, ok := r.Get("logger")
	if !ok {
		t.Fatal("expected plugin to be found")
	}

	if got != p {
		t.Fatal("expected registry to return the registered plugin")
	}
}

func TestRegistryRejectsDuplicatePlugin(t *testing.T) {
	r := NewRegistry()

	first := &registryTestPlugin{name: "logger", version: "1.0.0"}
	second := &registryTestPlugin{name: "logger", version: "2.0.0"}

	if err := r.Register(first); err != nil {
		t.Fatalf("unexpected first register error: %v", err)
	}

	err := r.Register(second)
	if !errors.Is(err, ErrDuplicatePlugin) {
		t.Fatalf("expected ErrDuplicatePlugin, got %v", err)
	}

	got, ok := r.Get("logger")
	if !ok {
		t.Fatal("expected original plugin to remain registered")
	}

	if got != first {
		t.Fatal("expected duplicate registration not to replace original plugin")
	}
}

func TestRegistryListPreservesRegistrationOrder(t *testing.T) {
	r := NewRegistry()

	plugins := []*registryTestPlugin{
		{name: "logger", version: "1.0.0"},
		{name: "metrics", version: "1.0.0"},
		{name: "health", version: "1.0.0"},
	}

	for _, p := range plugins {
		if err := r.Register(p); err != nil {
			t.Fatalf("unexpected register error: %v", err)
		}
	}

	got := r.List()

	if len(got) != len(plugins) {
		t.Fatalf("expected %d plugins, got %d", len(plugins), len(got))
	}

	for i, want := range plugins {
		if got[i] != want {
			t.Fatalf("unexpected plugin at index %d: got %v want %v", i, got[i], want)
		}
	}
}

func TestRegistryListReturnsSnapshot(t *testing.T) {
	r := NewRegistry()

	first := &registryTestPlugin{name: "first", version: "1.0.0"}
	second := &registryTestPlugin{name: "second", version: "1.0.0"}

	if err := r.Register(first); err != nil {
		t.Fatal(err)
	}

	if err := r.Register(second); err != nil {
		t.Fatal(err)
	}

	list := r.List()
	list[0] = second

	got, ok := r.Get("first")
	if !ok {
		t.Fatal("expected first plugin to remain registered")
	}

	if got != first {
		t.Fatal("List returned a live registry slice")
	}
}

func TestRegistryMissingPlugin(t *testing.T) {
	r := NewRegistry()

	if _, ok := r.Get("missing"); ok {
		t.Fatal("expected missing plugin lookup to return false")
	}
}

func TestRegistryRejectsNilPlugin(t *testing.T) {
	r := NewRegistry()

	if err := r.Register(nil); err == nil {
		t.Fatal("expected nil plugin registration to fail")
	}
}

func TestRegistryRejectsEmptyPluginName(t *testing.T) {
	r := NewRegistry()

	p := &registryTestPlugin{
		name:    "",
		version: "1.0.0",
	}

	if err := r.Register(p); err == nil {
		t.Fatal("expected empty plugin name registration to fail")
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	const workers = 20

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		i := i

		go func() {
			defer wg.Done()

			plugin := &registryTestPlugin{
				name:    "plugin-" + string(rune('a'+i)),
				version: "1.0.0",
			}

			if err := r.Register(plugin); err != nil {
				t.Errorf("unexpected register error: %v", err)
			}

			if _, ok := r.Get(plugin.Name()); !ok {
				t.Errorf("expected plugin %q to be found", plugin.Name())
			}
		}()
	}

	wg.Wait()

	if got := len(r.List()); got != workers {
		t.Fatalf("expected %d plugins, got %d", workers, got)
	}
}
