package app

// Add registers a module and returns any registration error.
func (a *App) Add(module Module) error {
	if module == nil {
		return ErrNilModule
	}

	if err := module.Register(a); err != nil {
		return err
	}

	a.mu.Lock()
	a.modules = append(a.modules, module)
	a.mu.Unlock()

	return nil
}

// Use registers a module.
//
// Use preserves the existing panic-on-registration-error behavior for
// backward compatibility. New integrations that need error propagation
// should use Add.
func (a *App) Use(module Module) *App {
	if err := a.Add(module); err != nil {
		panic(err)
	}

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
