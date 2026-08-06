package config

import (
	"sync"
	"time"
)

// Manager provides high-level configuration management operations.
type Manager struct {
	mu sync.RWMutex

	cfg Config

	watcher *Watcher

	observers []Observer
}

func NewManager(path string) (*Manager, error) {

	w, err := NewWatcher(path)
	if err != nil {
		return nil, err
	}

	return &Manager{
		cfg:       w.Config(),
		watcher:   w,
		observers: make([]Observer, 0),
	}, nil
}

func (m *Manager) Start() error {

	// mulai fsnotify watcher
	if err := m.watcher.Start(); err != nil {
		return err
	}

	// jalankan goroutine watcher
	m.watcher.Run()

	// sinkronisasi watcher -> manager
	go func() {

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {

			cfg := m.watcher.Config()

			m.mu.Lock()
			m.cfg = cfg
			m.mu.Unlock()

			m.notify()
		}

	}()

	return nil
}

func (m *Manager) Close() error {
	return m.watcher.Close()
}

func (m *Manager) Config() Config {

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cfg
}

func (m *Manager) Reload() error {

	cfg, err := Load(m.watcher.path)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()

	m.notify()

	return nil
}

func (m *Manager) Subscribe(o Observer) {

	m.mu.Lock()
	defer m.mu.Unlock()

	m.observers = append(m.observers, o)
}

func (m *Manager) notify() {

	m.mu.RLock()

	cfg := m.cfg
	observers := append([]Observer(nil), m.observers...)

	m.mu.RUnlock()

	for _, o := range observers {
		o.OnConfigReload(cfg)
	}
}
func (m *Manager) Save() error {

	m.mu.RLock()
	cfg := m.cfg
	m.mu.RUnlock()

	return Save(m.watcher.path, cfg)
}
