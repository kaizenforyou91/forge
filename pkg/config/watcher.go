package config

import (
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors configuration changes and reloads the latest configuration.
type Watcher struct {
	path string

	watcher *fsnotify.Watcher

	current Config

	mu sync.RWMutex

	observers []Observer
}

// NewWatcher creates a configuration watcher.
func NewWatcher(path string) (*Watcher, error) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		path:    path,
		watcher: watcher,
	}

	if err := w.Load(); err != nil {
		watcher.Close()
		return nil, err
	}

	return w, nil
}

// Start begins watching the configuration file for changes.
func (w *Watcher) Start() error {
	return w.watcher.Add(w.path)
}

// Close stops the watcher and releases all associated resources.
func (w *Watcher) Close() error {
	return w.watcher.Close()
}

// Events returns the underlying file system event channel.
func (w *Watcher) Events() <-chan fsnotify.Event {
	return w.watcher.Events
}

func (w *Watcher) Errors() <-chan error {
	return w.watcher.Errors
}

// Load reloads the configuration from disk and notifies all registered
// observers if loading succeeds.
func (w *Watcher) Load() error {

	cfg, err := Load(w.path)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.current = cfg

	w.notify(cfg)

	return nil
}

// Config returns the most recently loaded configuration.
//
// This method is safe for concurrent use.
func (w *Watcher) Config() Config {

	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.current
}

// Run starts the background goroutine that listens for file changes
// and automatically reloads the configuration.
func (w *Watcher) Run() {

	go func() {

		for {
			select {

			case event := <-w.Events():

				if event.Op&fsnotify.Write == fsnotify.Write {

					cfg, err := Load(w.path)
					if err != nil {
						continue
					}

					w.mu.Lock()
					w.current = cfg
					w.mu.Unlock()
				}

			case <-w.Errors():
				// Ignore watcher errors for now.
			}
		}

	}()
}

// Register registers an observer that will be notified whenever
// the configuration is reloaded.
func (w *Watcher) Register(o Observer) {
	w.observers = append(w.observers, o)
}
func (w *Watcher) notify(cfg Config) {

	for _, o := range w.observers {
		o.OnConfigReload(cfg)
	}

}
