package container

import "testing"

func TestParseConstructor(t *testing.T) {

	type Database struct{}

	type Repository struct{}

	fn := func(db *Database) *Repository {

		return &Repository{}

	}

	c, err := ParseConstructor(fn)

	if err != nil {

		t.Fatal(err)

	}

	if len(c.Input) != 1 {

		t.Fatal("invalid input count")

	}

	if c.Output == nil {

		t.Fatal("invalid output")

	}

}
