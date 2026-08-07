package app

import "testing"

type LifeModule struct{}

func (LifeModule) Name() string {
	return "life"
}

func (LifeModule) Register(app *App) error {
	return nil
}

func (LifeModule) Start(app *App) error {
	return nil
}

func (LifeModule) Stop(app *App) error {
	return nil
}

func TestLifecycle(t *testing.T) {

	var _ Lifecycle = LifeModule{}
}
