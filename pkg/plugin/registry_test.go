package plugin

import (
	"errors"
	"sync"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/config"
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

type registryUsePlugin struct {
	name       string
	registerFn func(*app.App) error
	started    bool
	stopped    bool
}

func (p *registryUsePlugin) Name() string {
	return p.name
}

func (p *registryUsePlugin) Version() string {
	return "1.0.0"
}

func (p *registryUsePlugin) Register(a *app.App) error {
	if p.registerFn != nil {
		return p.registerFn(a)
	}
	return nil
}

func (p *registryUsePlugin) Start(*app.App) error {
	p.started = true
	return nil
}

func (p *registryUsePlugin) Stop(*app.App) error {
	p.stopped = true
	return nil
}

func TestRegistryUseRegistersPluginsInOrder(t *testing.T) {
	r := NewRegistry()
	a := app.New()

	first := &registryUsePlugin{name: "first"}
	second := &registryUsePlugin{name: "second"}

	if err := r.Register(first); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(second); err != nil {
		t.Fatal(err)
	}

	if err := r.Use(a); err != nil {
		t.Fatal(err)
	}

	modules := a.Modules()
	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(modules))
	}

	if modules[0].Name() != "first" {
		t.Fatalf("expected first module to be first, got %q", modules[0].Name())
	}

	if modules[1].Name() != "second" {
		t.Fatalf("expected second module to be second, got %q", modules[1].Name())
	}

	if first.started || second.started {
		t.Fatal("Registry.Use must not start plugins")
	}

	if first.stopped || second.stopped {
		t.Fatal("Registry.Use must not stop plugins")
	}
}

func TestRegistryUsePropagatesRegistrationError(t *testing.T) {
	r := NewRegistry()
	a := app.New()

	first := &registryUsePlugin{name: "first"}

	wantErr := errors.New("plugin registration failed")
	second := &registryUsePlugin{
		name: "second",
		registerFn: func(*app.App) error {
			return wantErr
		},
	}

	if err := r.Register(first); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(second); err != nil {
		t.Fatal(err)
	}

	err := r.Use(a)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected registration error %v, got %v", wantErr, err)
	}

	modules := a.Modules()
	if len(modules) != 1 {
		t.Fatalf("expected only first plugin to be registered, got %d", len(modules))
	}

	if modules[0].Name() != "first" {
		t.Fatalf("expected first plugin to remain registered, got %q", modules[0].Name())
	}
}

type lifecyclePlugin struct {
	name        string
	order       *[]string
	startErr    error
	registerErr error
}

func (p *lifecyclePlugin) Name() string {
	return p.name
}

func (p *lifecyclePlugin) Version() string {
	return "1.0.0"
}

func (p *lifecyclePlugin) Register(*app.App) error {
	if p.registerErr != nil {
		return p.registerErr
	}

	return nil
}

func (p *lifecyclePlugin) Start(*app.App) error {
	*p.order = append(*p.order, p.name+":start")

	if p.startErr != nil {
		return p.startErr
	}

	return nil
}

func (p *lifecyclePlugin) Stop(*app.App) error {
	*p.order = append(*p.order, p.name+":stop")
	return nil
}

func TestRegistryPluginsFollowAppLifecycle(t *testing.T) {
	r := NewRegistry()
	a := app.New()

	var order []string

	first := &lifecyclePlugin{
		name:  "first",
		order: &order,
	}

	second := &lifecyclePlugin{
		name:  "second",
		order: &order,
	}

	if err := r.Register(first); err != nil {
		t.Fatal(err)
	}

	if err := r.Register(second); err != nil {
		t.Fatal(err)
	}

	if err := r.Use(a); err != nil {
		t.Fatal(err)
	}

	if err := a.Start(); err != nil {
		t.Fatal(err)
	}

	if !a.Started() {
		t.Fatal("expected app to be running")
	}

	expectedStart := []string{
		"first:start",
		"second:start",
	}

	if len(order) != len(expectedStart) {
		t.Fatalf("expected %d start events, got %d: %#v", len(expectedStart), len(order), order)
	}

	for i, want := range expectedStart {
		if order[i] != want {
			t.Fatalf("unexpected start order at %d: got %q want %q", i, order[i], want)
		}
	}

	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"first:start",
		"second:start",
		"second:stop",
		"first:stop",
	}

	if len(order) != len(expected) {
		t.Fatalf("expected lifecycle events %#v, got %#v", expected, order)
	}

	for i, want := range expected {
		if order[i] != want {
			t.Fatalf("unexpected lifecycle order at %d: got %q want %q", i, order[i], want)
		}
	}
}

func TestRegistryPluginStartFailureRollsBackPreviousPlugins(t *testing.T) {
	r := NewRegistry()
	a := app.New()

	var order []string
	startErr := errors.New("plugin start failed")

	first := &lifecyclePlugin{
		name:  "first",
		order: &order,
	}

	second := &lifecyclePlugin{
		name:     "second",
		order:    &order,
		startErr: startErr,
	}

	third := &lifecyclePlugin{
		name:  "third",
		order: &order,
	}

	if err := r.Register(first); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(second); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(third); err != nil {
		t.Fatal(err)
	}

	if err := r.Use(a); err != nil {
		t.Fatal(err)
	}

	err := a.Start()
	if !errors.Is(err, startErr) {
		t.Fatalf("expected start error %v, got %v", startErr, err)
	}

	if a.Started() {
		t.Fatal("expected app to remain stopped after failed start")
	}

	expected := []string{
		"first:start",
		"second:start",
		"first:stop",
	}

	if len(order) != len(expected) {
		t.Fatalf("expected lifecycle events %#v, got %#v", expected, order)
	}

	for i, want := range expected {
		if order[i] != want {
			t.Fatalf("unexpected lifecycle event at %d: got %q want %q", i, order[i], want)
		}
	}
}

func TestRegistryUseConfiguredSkipsDisabledPlugins(t *testing.T) {
	r := NewRegistry()
	a := app.New()
	cfg := config.Default()

	cfg.Plugins.Logger.Enabled = false

	disabled := &registryUsePlugin{
		name: "logger",
	}

	if err := r.Register(disabled); err != nil {
		t.Fatal(err)
	}

	if err := r.UseConfigured(a, cfg); err != nil {
		t.Fatal(err)
	}

	if len(a.Modules()) != 0 {
		t.Fatalf("expected no modules, got %d", len(a.Modules()))
	}
}

func TestRegistryUseConfiguredRegistersEnabledPlugins(t *testing.T) {
	r := NewRegistry()
	a := app.New()
	cfg := config.Default()

	cfg.Plugins.Logger.Enabled = true

	enabled := &registryUsePlugin{
		name: "logger",
	}

	if err := r.Register(enabled); err != nil {
		t.Fatal(err)
	}

	if err := r.UseConfigured(a, cfg); err != nil {
		t.Fatal(err)
	}

	modules := a.Modules()
	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}

	if modules[0].Name() != "logger" {
		t.Fatalf("expected logger module, got %q", modules[0].Name())
	}
}

type pluginService struct {
	Name string
}

type containerPlugin struct {
	service  *pluginService
	resolved *pluginService
}

func (p *containerPlugin) Name() string {
	return "container-plugin"
}

func (p *containerPlugin) Version() string {
	return "1.0.0"
}

func (p *containerPlugin) Register(a *app.App) error {
	p.service = &pluginService{Name: "from-plugin"}

	return a.Container().RegisterSingleton(p.service)
}

func (p *containerPlugin) Start(a *app.App) error {
	var resolved *pluginService

	if err := a.Container().Resolve(&resolved); err != nil {
		return err
	}

	p.resolved = resolved
	return nil
}

func (p *containerPlugin) Stop(*app.App) error {
	return nil
}

func TestPluginUsesAppContainer(t *testing.T) {
	r := NewRegistry()
	a := app.New()

	p := &containerPlugin{}

	if err := r.Register(p); err != nil {
		t.Fatal(err)
	}

	if err := r.Use(a); err != nil {
		t.Fatal(err)
	}

	if err := a.Start(); err != nil {
		t.Fatal(err)
	}

	if p.service == nil {
		t.Fatal("expected plugin to register service")
	}

	if p.resolved == nil {
		t.Fatal("expected plugin to resolve service")
	}

	if p.resolved != p.service {
		t.Fatal("expected resolved singleton to be the registered instance")
	}

	if p.resolved.Name != "from-plugin" {
		t.Fatalf("unexpected resolved service name: %q", p.resolved.Name)
	}

	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
}
