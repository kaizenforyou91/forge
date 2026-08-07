package app

// Run starts the application.
func (a *App) Run() error {

	if err := a.Start(); err != nil {
		return err
	}

	return nil
}
