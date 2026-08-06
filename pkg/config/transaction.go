package config

// Transaction performs atomic configuration updates.
type Transaction struct {
	path string
}

// NewTransaction creates a configuration transaction.
func NewTransaction(path string) *Transaction {
	return &Transaction{
		path: path,
	}
}

// Commit validates and saves a configuration atomically.
func (t *Transaction) Commit(cfg Config) error {

	// Before Save Hooks
	if err := RunBeforeSave(&cfg); err != nil {
		return err
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Backup
	if err := Backup(t.path); err != nil {
		return err
	}

	// Save
	if err := writeConfig(t.path, cfg); err != nil {
		_ = Restore(t.path)
		return err
	}

	// update cache setelah berhasil disimpan
	SetCache(cfg)

	// Verify
	if err := t.verify(); err != nil {
		_ = Restore(t.path)
		return err
	}

	// After Save Hooks
	if err := RunAfterSave(&cfg); err != nil {
		return err
	}

	return nil
}

func (t *Transaction) verify() error {

	_, err := Load(t.path)

	return err
}
