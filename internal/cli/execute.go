package cli

func Execute() error {
	return NewRootCommand().Execute()
}
