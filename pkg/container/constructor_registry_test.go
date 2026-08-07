package container

import "testing"

func TestRegisterConstructor(t *testing.T) {

	c := New()

	type Database struct{}

	type Repository struct{}

	fn := func(db *Database) *Repository {
		return &Repository{}
	}

	if err := c.RegisterConstructor(fn); err != nil {
		t.Fatal(err)
	}

	if c.ConstructorCount() != 1 {
		t.Fatalf("expected 1 constructor, got %d", c.ConstructorCount())
	}
}
func TestRegisterInvalidConstructor(t *testing.T) {

	c := New()

	err := c.RegisterConstructor(123)

	if err == nil {
		t.Fatal("expected invalid constructor error")
	}
}
