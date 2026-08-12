package app

// Use registers a module.
func (a *App) Use(module Module) *App {
	if err := module.Register(a); err != nil {
		panic(err)
	}

	a.mu.Lock()
	a.modules = append(a.modules, module)
	a.mu.Unlock()

	return a
}

// HasModule reports whether a module with the given name exists.
func (a *App) HasModule(name string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, module := range a.modules {
		if module.Name() == name {
			return true
		}
	}

	return false
}
