package tool

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func mustCatalog(t *testing.T, definitions ...Definition) *Catalog {
	t.Helper()
	catalog, err := NewCatalog(definitions)
	if err != nil || catalog == nil {
		t.Fatalf("NewCatalog: catalog nil=%v, error=%v", catalog == nil, err)
	}
	return catalog
}

func assertDecision(t *testing.T, catalog *Catalog, call Call, want Reason) Admission {
	t.Helper()
	got, err := catalog.Admit(call)
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	wantStatus := Rejected
	if want == Allowed {
		wantStatus = Admitted
	}
	if got.Status() != wantStatus || got.Reason() != want {
		t.Fatalf("decision: %v; want status=%d reason=%d", got, wantStatus, want)
	}
	if want != Allowed {
		if exposed, ok := got.AdmittedCall(); ok || exposed.Name != "" || exposed.Arguments != nil {
			t.Fatal("rejection exposed a call")
		}
		if got.call.Name != "" || got.call.Arguments != nil {
			t.Fatal("rejection retained a call")
		}
	}
	return got
}

func TestCatalogEmptyAndNil(t *testing.T) {
	for name, definitions := range map[string][]Definition{
		"nil definitions":   nil,
		"empty definitions": {},
	} {
		t.Run(name, func(t *testing.T) {
			assertDecision(t, mustCatalog(t, definitions...), Call{Name: "tool", Arguments: []byte("{}")}, UnknownTool)
		})
	}
	var zero Catalog
	assertDecision(t, &zero, Call{Name: "tool"}, UnknownTool)
	var absent *Catalog
	got, err := absent.Admit(Call{})
	if got.Status() != Rejected || got.Reason() != NotEvaluated || !errors.Is(err, ErrInvalidCatalog) {
		t.Fatalf("nil receiver: %v, %v", got, err)
	}
	if call, ok := got.AdmittedCall(); ok || call.Name != "" || call.Arguments != nil || got.call.Arguments != nil {
		t.Fatal("nil receiver exposed or retained a call")
	}
	if errors.Unwrap(err) != nil || err.Error() != "ai/tool: invalid catalog" {
		t.Fatal("catalog error exposed a raw cause")
	}
}

func TestCatalogNamesAndTypes(t *testing.T) {
	valid := []string{"a", "_", "A0_-", strings.Repeat("a", 64)}
	for _, name := range valid {
		t.Run("valid_"+name, func(t *testing.T) {
			c := mustCatalog(t, Definition{Name: name, Parameters: []Parameter{{Name: name, Type: String}}})
			assertDecision(t, c, Call{Name: name, Arguments: []byte("{}")}, Allowed)
		})
	}
	invalid := []string{"", strings.Repeat("a", 65), "0tool", "-tool", "a b", " a", "a ", "a\t", "a\n", "a\x00", "é", "a.b", "*", "a*"}
	for i, name := range invalid {
		t.Run(fmt.Sprintf("invalid_%d", i), func(t *testing.T) {
			for _, defs := range [][]Definition{
				{{Name: name}},
				{{Name: "valid", Parameters: []Parameter{{Name: name, Type: String}}}},
			} {
				got, err := NewCatalog(defs)
				if got != nil || !errors.Is(err, ErrInvalidCatalog) {
					t.Fatal("invalid identity did not invalidate entire catalog")
				}
			}
		})
	}
	for _, kind := range []ValueType{InvalidType, ValueType(7), ValueType(255)} {
		got, err := NewCatalog([]Definition{{Name: "valid", Parameters: []Parameter{{Name: "p", Type: kind}}}})
		if got != nil || !errors.Is(err, ErrInvalidCatalog) {
			t.Fatalf("invalid type %d accepted", kind)
		}
	}
	for _, kind := range []ValueType{String, Number, Boolean, Object, Array, Null} {
		mustCatalog(t, Definition{Name: "valid", Parameters: []Parameter{{Name: "p", Type: kind}}})
	}
}

func TestCatalogDuplicatesInvalidateAll(t *testing.T) {
	cases := [][]Definition{
		{{Name: "first"}, {Name: "same"}, {Name: "same"}},
		{{Name: "first"}, {Name: "last", Parameters: []Parameter{
			{Name: "p", Type: String}, {Name: "p", Type: Number},
		}}},
		{{Name: "last", Parameters: []Parameter{
			{Name: "p", Type: String}, {Name: "p", Type: String},
		}}},
	}
	for i, defs := range cases {
		got, err := NewCatalog(defs)
		if got != nil || !errors.Is(err, ErrInvalidCatalog) {
			t.Fatalf("case %d: partially valid catalog published", i)
		}
	}
	// Case-sensitive definitions are distinct, not duplicate aliases.
	mustCatalog(t, Definition{Name: "Tool"}, Definition{Name: "tool"})
	mustCatalog(t, Definition{Name: "tool", Parameters: []Parameter{{Name: "P", Type: String}, {Name: "p", Type: String}}})
}

func TestCatalogCountBounds(t *testing.T) {
	defs := make([]Definition, 64)
	for i := range defs {
		defs[i] = Definition{Name: fmt.Sprintf("tool%d", i)}
	}
	c := mustCatalog(t, defs...)
	assertDecision(t, c, Call{Name: "tool63", Arguments: []byte("{}")}, Allowed)
	if got, err := NewCatalog(append(defs, Definition{Name: "extra"})); got != nil || !errors.Is(err, ErrInvalidCatalog) {
		t.Fatal("65 tools accepted")
	}
	params := make([]Parameter, 32)
	for i := range params {
		params[i] = Parameter{Name: fmt.Sprintf("p%d", i), Type: Null}
	}
	c = mustCatalog(t, Definition{Name: "tool", Parameters: params})
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{}")}, Allowed)
	params = append(params, Parameter{Name: "extra", Type: Null})
	if got, err := NewCatalog([]Definition{{Name: "tool", Parameters: params}}); got != nil || !errors.Is(err, ErrInvalidCatalog) {
		t.Fatal("33 parameters accepted")
	}
}

func TestCatalogOwnsConstructionInputs(t *testing.T) {
	params := []Parameter{{Name: "text", Type: String, Required: true}}
	defs := []Definition{{Name: "summarize", Parameters: params}}
	c := mustCatalog(t, defs...)
	params[0] = Parameter{Name: "replacement", Type: Number}
	defs[0].Name = "replacement"
	defs[0].Parameters = nil
	assertDecision(t, c, Call{Name: "summarize", Arguments: []byte("{\"text\":\"dummy\"}")}, Allowed)
	assertDecision(t, c, Call{Name: "summarize", Arguments: []byte("{}")}, InvalidArguments)
	assertDecision(t, c, Call{Name: "replacement", Arguments: []byte("{}")}, UnknownTool)
}
