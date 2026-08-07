package container

import "testing"

type ModuleConfig struct {
	Name string
}

type TestModule struct{}

func (TestModule) Register(b *Builder) {

	b.Singleton(&ModuleConfig{
		Name: "Forge",
	})

}

func TestModuleRegistration(t *testing.T) {

	c := NewBuilder().
		Module(TestModule{}).
		Build()

	cfg, err := Make[*ModuleConfig](c)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Name != "Forge" {
		t.Fatal("unexpected value")
	}
}
