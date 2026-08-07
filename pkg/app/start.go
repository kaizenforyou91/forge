package app

// Start starts the application.
func (a *App) Start() error {

	if a.started {
		return nil
	}

	for _, module := range a.modules {

		if err := module.Start(a); err != nil {
			return err
		}

	}

	a.started = true

	return nil
}
