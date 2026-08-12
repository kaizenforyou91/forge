package app

import "errors"

// Stop stops the application.
func (a *App) Stop() error {
	proceed, err := a.runtime.BeginStop()
	if err != nil {
		return err
	}

	if !proceed {
		return nil
	}

	a.Cancel()

	a.mu.RLock()
	modules := append([]Module(nil), a.modules...)
	a.mu.RUnlock()

	var stopErrors []error

	for i := len(modules) - 1; i >= 0; i-- {
		if err := modules[i].Stop(a); err != nil {
			stopErrors = append(stopErrors, err)
		}
	}

	a.runtime.SetStopped()

	if len(stopErrors) == 0 {
		return nil
	}

	return errors.Join(stopErrors...)
}
