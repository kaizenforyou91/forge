package container

import "testing"

func TestNewContainer(t *testing.T) {

	c := New()

	if c == nil {
		t.Fatal("container is nil")
	}

	if c.services == nil {
		t.Fatal("services map is nil")
	}

}
func TestRegister(t *testing.T) {

	c := New()

	type Service struct {
		Name string
	}

	s := Service{
		Name: "Forge",
	}

	if err := c.Register(s); err != nil {
		t.Fatal(err)
	}

	if c.Count() != 1 {
		t.Fatalf("expected 1 service, got %d", c.Count())
	}

}
func TestResolve(t *testing.T) {

	c := New()

	type Service struct {
		Name string
	}

	original := Service{
		Name: "Forge",
	}

	if err := c.Register(original); err != nil {
		t.Fatal(err)
	}

	var loaded Service

	if err := c.Resolve(&loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.Name != "Forge" {
		t.Fatalf("expected Forge, got %s", loaded.Name)
	}

}
func TestResolveNotFound(t *testing.T) {

	c := New()

	type Service struct {
		Name string
	}

	var s Service

	err := c.Resolve(&s)

	if err != ErrServiceNotFound {
		t.Fatal("expected ErrServiceNotFound")
	}

}
func TestResolveInvalidTarget(t *testing.T) {

	c := New()

	type Service struct{}

	var s Service

	err := c.Resolve(s)

	if err != ErrInvalidTarget {
		t.Fatal("expected ErrInvalidTarget")
	}

}
func TestRegisterSingleton(t *testing.T) {

	c := New()

	type Service struct{}

	if err := c.RegisterSingleton(Service{}); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterTransient(t *testing.T) {

	c := New()

	type Service struct{}

	if err := c.RegisterTransient(Service{}); err != nil {
		t.Fatal(err)
	}
}
func TestRegisterFactory(t *testing.T) {

	c := New()

	type Service struct {
		Name string
	}

	err := c.RegisterFactory(func() any {
		return Service{Name: "Forge"}
	})

	if err != nil {
		t.Fatal(err)
	}

	if c.Count() != 1 {
		t.Fatal("expected one service")
	}
}
func TestResolveFactory(t *testing.T) {

	c := New()

	type Service struct {
		Name string
	}

	err := c.RegisterFactory(func() any {
		return Service{Name: "Forge"}
	})

	if err != nil {
		t.Fatal(err)
	}

	var svc Service

	if err := c.Resolve(&svc); err != nil {
		t.Fatal(err)
	}

	if svc.Name != "Forge" {
		t.Fatal("invalid service")
	}
}
