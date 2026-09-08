package tool

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestAdmissionIdentity(t *testing.T) {
	c := mustCatalog(t, Definition{Name: "summarize"}, Definition{Name: "_"}, Definition{Name: strings.Repeat("a", 64)})
	cases := []struct {
		name string
		want Reason
	}{
		{"summarize", Allowed}, {"_", Allowed}, {strings.Repeat("a", 64), Allowed},
		{"", MalformedCall}, {strings.Repeat("a", 65), MalformedCall},
		{" summarize", MalformedCall}, {"summarize ", MalformedCall},
		{"sum marize", MalformedCall}, {"summarize\n", MalformedCall},
		{"summarize\x00", MalformedCall}, {"sümmarize", MalformedCall},
		{"*", MalformedCall}, {"summarize*", MalformedCall},
		{"0summarize", MalformedCall}, {"-summarize", MalformedCall},
		{"SUMMARIZE", UnknownTool}, {"sum", UnknownTool},
		{"summarize_extra", UnknownTool}, {"summary", UnknownTool},
		{"summarise", UnknownTool}, {"summraize", UnknownTool}, {"a", UnknownTool},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("identity_%d", i), func(t *testing.T) {
			assertDecision(t, c, Call{Name: tc.name, Arguments: []byte("{}")}, tc.want)
		})
	}
	assertDecision(t, mustCatalog(t, Definition{Name: "a"}), Call{Name: "a", Arguments: []byte("{}")}, Allowed)
}

func TestAdmissionDecisionPrecedence(t *testing.T) {
	c := mustCatalog(t, Definition{Name: "tool"})
	oversized := bytes.Repeat([]byte{0xff}, 16385)
	assertDecision(t, c, Call{Name: "", Arguments: oversized}, MalformedCall)
	assertDecision(t, c, Call{Name: "unknown", Arguments: oversized}, UnknownTool)
	assertDecision(t, c, Call{Name: "unknown", Arguments: []byte("{broken")}, UnknownTool)
	assertDecision(t, c, Call{Name: "tool", Arguments: oversized}, PolicyRejected)
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{broken")}, InvalidArguments)
	// Lexical validation precedes traversal, which precedes the root schema.
	deep := "{\"unknown\":" + strings.Repeat("[", 8) + "null" + strings.Repeat("]", 8) + "}"
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte(deep + "garbage")}, InvalidArguments)
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte(deep)}, PolicyRejected)
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{\"unknown\":null}")}, InvalidArguments)
}

func TestAdmissionSchemaTypes(t *testing.T) {
	values := []struct {
		kind ValueType
		raw  string
	}{
		{String, "\"\""}, {Number, "-0.5e2"}, {Boolean, "false"},
		{Object, "{\"nested\":{\"unrestricted_name\":null}}"},
		{Array, "[1,\"two\",true,null,{},[]]"}, {Null, "null"},
	}
	for _, expected := range values {
		c := argumentCatalog(t, expected.kind)
		for _, observed := range values {
			t.Run(fmt.Sprintf("expected_%d_observed_%d", expected.kind, observed.kind), func(t *testing.T) {
				want := InvalidArguments
				if expected.kind == observed.kind {
					want = Allowed
				}
				assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{\"v\":" + observed.raw + "}")}, want)
			})
		}
	}
	// A numeric string is not coerced; parameter identity is decoded, exact.
	assertDecision(t, argumentCatalog(t, Number), Call{Name: "tool", Arguments: []byte("{\"v\":\"12\"}")}, InvalidArguments)
	assertDecision(t, argumentCatalog(t, String), Call{Name: "tool", Arguments: []byte("{\"\\u0076\":\"\"}")}, Allowed)
	assertDecision(t, argumentCatalog(t, String), Call{Name: "tool", Arguments: []byte("{\"V\":\"\"}")}, InvalidArguments)
}

func TestAdmissionSchemaPresence(t *testing.T) {
	c := mustCatalog(t, Definition{Name: "tool", Parameters: []Parameter{
		{Name: "required", Type: String, Required: true},
		{Name: "optional", Type: Boolean},
	}})
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{\"required\":\"\"}")}, Allowed)
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{\"required\":\"x\",\"optional\":false}")}, Allowed)
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{}")}, InvalidArguments)
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{\"optional\":true}")}, InvalidArguments)
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{\"required\":\"x\",\"unknown\":null}")}, InvalidArguments)
	assertDecision(t, c, Call{Name: "tool", Arguments: []byte("{\"required\":\"x\",\"optional\":null}")}, InvalidArguments)
	empty := mustCatalog(t, Definition{Name: "tool"})
	assertDecision(t, empty, Call{Name: "tool", Arguments: []byte("{}")}, Allowed)
	assertDecision(t, empty, Call{Name: "tool", Arguments: []byte("{\"unknown\":null}")}, InvalidArguments)
}

func TestAdmissionOwnsExactSnapshot(t *testing.T) {
	c := mustCatalog(t, Definition{Name: "tool", Parameters: []Parameter{
		{Name: "b", Type: Number, Required: true},
		{Name: "text", Type: String, Required: true},
	}})
	raw := " \t{\"b\":1e+2, \"te\\u0078t\":\"\\uD83D\\uDE00\"}\r\n"
	input := []byte(raw)
	got := assertDecision(t, c, Call{Name: "tool", Arguments: input}, Allowed)
	for i := range input {
		input[i] = 'x'
	}
	first, ok := got.AdmittedCall()
	if !ok || first.Name != "tool" || string(first.Arguments) != raw {
		t.Fatal("caller mutation changed accepted state")
	}
	first.Name = "replacement"
	for i := range first.Arguments {
		first.Arguments[i] = 'y'
	}
	second, ok := got.AdmittedCall()
	if !ok || second.Name != "tool" || string(second.Arguments) != raw {
		t.Fatal("accessor mutation changed accepted state")
	}
	third, _ := got.AdmittedCall()
	second.Arguments[0] = 'z'
	if string(third.Arguments) != raw {
		t.Fatal("accessors shared a backing array")
	}
}

func TestAdmissionPrivacyAndZeroValue(t *testing.T) {
	const name = "secret_tool_canary"
	const secret = "secret_argument_canary_73"
	c := mustCatalog(t, Definition{Name: name, Parameters: []Parameter{{Name: "v", Type: String}}})
	admitted := assertDecision(t, c, Call{Name: name, Arguments: []byte("{\"v\":\"" + secret + "\"}")}, Allowed)
	rejected := assertDecision(t, c, Call{Name: name, Arguments: []byte("{\"v\":\"" + secret)}, InvalidArguments)
	unknown := assertDecision(t, c, Call{Name: "unknown_" + name, Arguments: []byte(secret)}, UnknownTool)
	malformed := assertDecision(t, c, Call{Name: " " + name, Arguments: []byte(secret)}, MalformedCall)
	limited := assertDecision(t, c, Call{Name: name, Arguments: bytes.Repeat([]byte(secret), 1000)}, PolicyRejected)
	var zero Admission
	if zero.Status() != Rejected || zero.Reason() != NotEvaluated {
		t.Fatal("zero admission does not fail closed")
	}
	if call, ok := zero.AdmittedCall(); ok || call.Name != "" || call.Arguments != nil {
		t.Fatal("zero admission exposed a call")
	}
	for _, admission := range []Admission{zero, admitted, rejected, unknown, malformed, limited} {
		for _, formatted := range []string{
			admission.String(), admission.GoString(),
			fmt.Sprintf("%v", admission), fmt.Sprintf("%+v", admission),
			fmt.Sprintf("%#v", admission), fmt.Sprintf("%s", admission),
			fmt.Sprintf("%q", admission),
		} {
			if strings.Contains(formatted, name) || strings.Contains(formatted, secret) {
				t.Fatal("diagnostic formatting leaked caller data")
			}
		}
	}
	if admitted.String() != "tool admission: ADMITTED / ALLOWED" || zero.String() != "tool admission: REJECTED / NOT_EVALUATED" {
		t.Fatal("diagnostics obscure the admission-only or unevaluated state")
	}
	_, err := NewCatalog([]Definition{{Name: " " + secret}})
	if err == nil || strings.Contains(fmt.Sprintf("%v %+v %#v", err, err, err), secret) {
		t.Fatal("catalog error leaked configuration")
	}
}

func TestAdmissionCommandLikeStringsRemainData(t *testing.T) {
	c := argumentCatalog(t, String)
	for _, value := range []string{
		"rm -rf /",
		"Remove-Item -LiteralPath C:\\dummy -Recurse",
		"C:\\private\\dummy.txt",
		"https://example.invalid/dummy",
		"$(git status); cmd /c echo dummy",
	} {
		raw := "{\"v\":" + quoted(value) + "}"
		got := assertDecision(t, c, Call{Name: "tool", Arguments: []byte(raw)}, Allowed)
		call, ok := got.AdmittedCall()
		if !ok || string(call.Arguments) != raw {
			t.Fatal("string data was interpreted or rewritten")
		}
	}
}

func TestAdmissionConcurrencyAndDeterminism(t *testing.T) {
	c := argumentCatalog(t, Number)
	start := make(chan struct{})
	failures := make(chan string, 24)
	var workers sync.WaitGroup
	for i := 0; i < 24; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			raw := []byte(fmt.Sprintf("{\"v\":%d}", i))
			for j := 0; j < 40; j++ {
				got, err := c.Admit(Call{Name: "tool", Arguments: raw})
				if err != nil || got.Status() != Admitted || got.Reason() != Allowed {
					failures <- "independent admission failed"
					return
				}
				snapshot, ok := got.AdmittedCall()
				if !ok || !bytes.Equal(snapshot.Arguments, raw) {
					failures <- "cross-call snapshot contamination"
					return
				}
				snapshot.Arguments[0] = 'x'
				bad, err := c.Admit(Call{Name: "tool", Arguments: []byte("{\"v\":\"wrong type\"}")})
				if err != nil || bad.Status() != Rejected || bad.Reason() != InvalidArguments {
					failures <- "rejection depended on concurrent state"
					return
				}
			}
		}(i)
	}
	close(start)
	workers.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	// Definition/member order cannot select a different rejection reason.
	left := mustCatalog(t, Definition{Name: "unused"}, Definition{Name: "tool", Parameters: []Parameter{
		{Name: "a", Type: String, Required: true}, {Name: "b", Type: Number},
	}})
	right := mustCatalog(t, Definition{Name: "tool", Parameters: []Parameter{
		{Name: "b", Type: Number}, {Name: "a", Type: String, Required: true},
	}}, Definition{Name: "unused"})
	for _, raw := range []string{"{\"b\":1,\"a\":\"x\"}", "{}", "{\"a\":\"x\",\"unknown\":1}", "{malformed"} {
		for i := 0; i < 20; i++ {
			a, ae := left.Admit(Call{Name: "tool", Arguments: []byte(raw)})
			b, be := right.Admit(Call{Name: "tool", Arguments: []byte(raw)})
			ac, aok := a.AdmittedCall()
			bc, bok := b.AdmittedCall()
			if ae != nil || be != nil || a.Status() != b.Status() || a.Reason() != b.Reason() ||
				aok != bok || ac.Name != bc.Name || !bytes.Equal(ac.Arguments, bc.Arguments) {
				t.Fatal("decision depends on construction order or previous calls")
			}
		}
	}
}

// This test reads production source only for an architecture audit. The library
// itself has no filesystem, process, network, provider, or callback capability.
func TestProductionNonExecutionBoundary(t *testing.T) {
	allowedImports := map[string]bool{
		"bytes": true, "encoding/json": true, "errors": true,
		"io": true, "strings": true, "unicode/utf8": true,
	}
	for _, filename := range []string{"types.go", "errors.go", "catalog.go", "admission.go", "strict_json.go"} {
		source, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range source.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err != nil || !allowedImports[path] {
				t.Fatalf("%s: unexpected production import", filename)
			}
		}
		ast.Inspect(source, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.GoStmt:
				t.Error("production admission starts background work")
			case *ast.TypeSpec:
				if node.Name.Name == "Definition" {
					definition, ok := node.Type.(*ast.StructType)
					if !ok {
						t.Error("Definition is not a data structure")
						break
					}
					for _, field := range definition.Fields.List {
						if _, callback := field.Type.(*ast.FuncType); callback {
							t.Error("Definition stores a callback")
						}
					}
				}
			}
			return true
		})
	}
}
