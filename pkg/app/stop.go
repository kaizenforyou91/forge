package app

// Stop stops the application.
func (a *App) Stop() error {

	if !a.started {
		return nil
	}

	// Stop modules in reverse order.
	for i := len(a.modules) - 1; i >= 0; i-- {

		if err := a.modules[i].Stop(a); err != nil {
			return err
		}

	}

	a.Cancel()

	a.started = false

	return nil
}
