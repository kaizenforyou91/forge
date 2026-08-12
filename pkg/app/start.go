package app

// Start starts the application.
func (a *App) Start() error {
	proceed, err := a.runtime.BeginStart()
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	a.mu.RLock()
	modules := append([]Module(nil), a.modules...)
	a.mu.RUnlock()

	started := make([]Module, 0, len(modules))

	for _, module := range modules {
		if err := module.Start(a); err != nil {
			for i := len(started) - 1; i >= 0; i-- {
				_ = started[i].Stop(a)
			}
			a.runtime.Cancel()
			a.runtime.SetStopped()
			return err
		}

		started = append(started, module)
	}

	a.runtime.SetRunning()

	return nil
}
