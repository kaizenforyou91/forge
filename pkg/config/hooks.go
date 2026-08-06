package config

// Hook represents a callback executed during configuration lifecycle events.
type Hook func(*Config) error

// Hooks contains all registered configuration lifecycle hooks.
type Hooks struct {
	BeforeLoad     []Hook
	AfterLoad      []Hook
	BeforeSave     []Hook
	AfterSave      []Hook
	BeforeValidate []Hook
	AfterValidate  []Hook
}

var hooks Hooks

func RegisterBeforeSave(h Hook) {
	hooks.BeforeSave = append(hooks.BeforeSave, h)
}

func RegisterAfterSave(h Hook) {
	hooks.AfterSave = append(hooks.AfterSave, h)
}

func RegisterBeforeLoad(h Hook) {
	hooks.BeforeLoad = append(hooks.BeforeLoad, h)
}

func RegisterAfterLoad(h Hook) {
	hooks.AfterLoad = append(hooks.AfterLoad, h)
}
func runHooks(list []Hook, cfg *Config) error {

	for _, h := range list {

		if err := h(cfg); err != nil {
			return err
		}
	}

	return nil
}

func RunBeforeLoad(cfg *Config) error {
	return runHooks(hooks.BeforeLoad, cfg)
}

func RunAfterLoad(cfg *Config) error {
	return runHooks(hooks.AfterLoad, cfg)
}

func RunBeforeSave(cfg *Config) error {
	return runHooks(hooks.BeforeSave, cfg)
}

func RunAfterSave(cfg *Config) error {
	return runHooks(hooks.AfterSave, cfg)
}

func RunBeforeValidate(cfg *Config) error {
	return runHooks(hooks.BeforeValidate, cfg)
}

func RunAfterValidate(cfg *Config) error {
	return runHooks(hooks.AfterValidate, cfg)
}

func RegisterBeforeValidate(h Hook) {
	hooks.BeforeValidate = append(hooks.BeforeValidate, h)
}

func RegisterAfterValidate(h Hook) {
	hooks.AfterValidate = append(hooks.AfterValidate, h)
}
