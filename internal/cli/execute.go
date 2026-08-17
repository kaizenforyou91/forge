package cli

// Execute executes the Forge CLI root command.
func Execute() error {
	return NewRootCommand().Execute()
}
