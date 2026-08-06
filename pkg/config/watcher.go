package config

import (
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	path string

	watcher *fsnotify.Watcher

	current Config

	mu sync.RWMutex

	observers []Observer
}

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

func (w *Watcher) Start() error {
	return w.watcher.Add(w.path)
}

func (w *Watcher) Close() error {
	return w.watcher.Close()
}

func (w *Watcher) Events() <-chan fsnotify.Event {
	return w.watcher.Events
}

func (w *Watcher) Errors() <-chan error {
	return w.watcher.Errors
}

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

func (w *Watcher) Config() Config {

	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.current
}

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
				// abaikan untuk tahap ini
			}
		}

	}()
}
func (w *Watcher) Register(o Observer) {
	w.observers = append(w.observers, o)
}
func (w *Watcher) notify(cfg Config) {

	for _, o := range w.observers {
		o.OnConfigReload(cfg)
	}

}
