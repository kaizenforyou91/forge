package config

import "testing"

type testObserver struct {
	called bool
}

func (o *testObserver) OnConfigReload(cfg Config) {
	o.called = true
}

func TestRegisterObserver(t *testing.T) {

	w := &Watcher{}

	obs := &testObserver{}

	w.Register(obs)

	if len(w.observers) != 1 {
		t.Fatal("observer not registered")
	}

	w.notify(Default())

	if !obs.called {
		t.Fatal("observer was not notified")
	}
}
