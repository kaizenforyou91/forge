package app

// Builder builds an application host.
type Builder struct {
	app *App
}

// NewBuilder creates a new Builder.
func NewBuilder() *Builder {

	return &Builder{
		app: New(),
	}
}

// Build returns the configured application.
func (b *Builder) Build() *App {

	return b.app

}
