package config

// Migration represents a configuration version migration.
type Migration struct {
	From int
	To   int
	Run  func(*Config) error
}

var migrations []Migration

// RegisterMigration registers a migration.
func RegisterMigration(m Migration) {
	migrations = append(migrations, m)
}

// Migrate executes pending migrations.
func Migrate(cfg *Config) error {

	for _, m := range migrations {

		if cfg.Version == m.From {

			if err := m.Run(cfg); err != nil {
				return err
			}

			cfg.Version = m.To
		}
	}

	return nil
}
