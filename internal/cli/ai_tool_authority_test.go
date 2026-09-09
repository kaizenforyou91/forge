package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

func runtimeToolAdmission(t *testing.T, definitions []tool.Definition, call tool.Call) tool.Admission {
	t.Helper()
	catalog, err := tool.NewCatalog(definitions)
	if err != nil {
		t.Fatal(err)
	}
	admission, err := catalog.Admit(call)
	if err != nil {
		t.Fatal(err)
	}
	return admission
}

func runtimeToolOutput(t *testing.T, authority *tool.Authority) string {
	t.Helper()
	admission := runtimeToolAdmission(t, authority.Definitions(), tool.Call{Name: "forge_runtime_info", Arguments: []byte(`{}`)})
	if admission.Status() != tool.Admitted || admission.Reason() != tool.Allowed {
		t.Fatal("runtime call not admitted")
	}
	result, err := authority.Executor().Execute(context.Background(), admission)
	if err != nil {
		t.Fatal(err)
	}
	return result.Output()
}

func TestAIRuntimeToolAuthority(t *testing.T) {
	metadata := aiRuntimeMetadata{Version: "private_version", Commit: "private_commit", BuildTime: "private_build_time"}
	authority, err := newAIRuntimeToolAuthority(metadata)
	if err != nil || authority == nil || authority.Executor() == nil {
		t.Fatal("construction failed", err)
	}
	definitions := authority.Definitions()
	if len(definitions) != 1 || definitions[0].Name != "forge_runtime_info" || len(definitions[0].Parameters) != 0 {
		t.Fatal("unexpected tool definition")
	}
	want := `{"version":"private_version","commit":"private_commit","build_time":"private_build_time"}`
	metadata.Version, metadata.Commit, metadata.BuildTime = "changed", "changed", "changed"
	for range 2 {
		output := runtimeToolOutput(t, authority)
		if output != want {
			t.Fatal("metadata snapshot or deterministic output changed")
		}
		decoder := json.NewDecoder(strings.NewReader(output))
		var object map[string]string
		if decoder.Decode(&object) != nil || !reflect.DeepEqual(object, map[string]string{"version": "private_version", "commit": "private_commit", "build_time": "private_build_time"}) {
			t.Fatal("incorrect runtime JSON object")
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			t.Fatal("more than one JSON value")
		}
	}
	// A returned definition cannot widen or replace the bundled authority.
	definitions[0].Name = "fake_tool"
	definitions[0].Parameters = []tool.Parameter{{Name: "extra", Type: tool.String}}
	if fresh := authority.Definitions(); len(fresh) != 1 || fresh[0].Name != "forge_runtime_info" || len(fresh[0].Parameters) != 0 {
		t.Fatal("returned definitions mutated authority")
	}
	if runtimeToolOutput(t, authority) != want {
		t.Fatal("executor changed after mutation")
	}
	fake := runtimeToolAdmission(t, definitions, tool.Call{Name: "fake_tool", Arguments: []byte(`{}`)})
	if fake.Status() != tool.Admitted {
		t.Fatal("fake admission fixture failed")
	}
	result, err := authority.Executor().Execute(context.Background(), fake)
	if err != tool.ErrExecutionDenied || result.Output() != "" {
		t.Fatal("fake tool executed")
	}
	for _, value := range []any{authority, *authority, authority.Executor()} {
		for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			text := fmt.Sprintf(format, value)
			for _, secret := range []string{"private_version", "private_commit", "private_build_time"} {
				if strings.Contains(text, secret) {
					t.Fatal("diagnostics exposed metadata")
				}
			}
		}
	}
}

// Both C1 and the runtime handler check Err before dispatch/return. Zero Err
// observations on a denied call prove that the handler's first statement was
// never reached, without exposing or replacing Authority's private handler.
type runtimeToolObservedContext struct {
	context.Context
	checks int
}

func (c *runtimeToolObservedContext) Err() error { c.checks++; return c.Context.Err() }

func TestAIRuntimeToolRejectsArgumentsAndCanceledContext(t *testing.T) {
	authority, err := newAIRuntimeToolAuthority(aiRuntimeMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	call := tool.Call{Name: "forge_runtime_info", Arguments: []byte(`{"extra":"x"}`)}
	rejected := runtimeToolAdmission(t, authority.Definitions(), call)
	if rejected.Status() != tool.Rejected || rejected.Reason() != tool.InvalidArguments {
		t.Fatal("unknown argument admitted")
	}
	// Also prove C1 rejects a call admitted by an independently wider catalog.
	wider := runtimeToolAdmission(t, []tool.Definition{{Name: "forge_runtime_info", Parameters: []tool.Parameter{{Name: "extra", Type: tool.String}}}}, call)
	if wider.Status() != tool.Admitted {
		t.Fatal("wider fixture failed")
	}
	for _, admission := range []tool.Admission{rejected, wider} {
		ctx := &runtimeToolObservedContext{Context: context.Background()}
		result, err := authority.Executor().Execute(ctx, admission)
		if err != tool.ErrExecutionDenied || result.Output() != "" || ctx.checks != 0 {
			t.Fatal("denied call reached handler")
		}
	}
	valid := runtimeToolAdmission(t, authority.Definitions(), tool.Call{Name: "forge_runtime_info", Arguments: []byte(`{}`)})
	observed := &runtimeToolObservedContext{Context: context.Background()}
	if _, err := authority.Executor().Execute(observed, valid); err != nil || observed.checks != 3 {
		t.Fatal("handler context check missing")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := authority.Executor().Execute(ctx, valid)
	if err != context.Canceled || result.Output() != "" {
		t.Fatal("canceled call produced output")
	}
}

func TestDefaultAIRuntimeToolAuthoritySnapshot(t *testing.T) {
	// No t.Parallel: build metadata globals are temporarily replaced in-process.
	oldVersion, oldCommit, oldBuildTime := AppVersion, Commit, BuildTime
	restore := func() { AppVersion, Commit, BuildTime = oldVersion, oldCommit, oldBuildTime }
	t.Cleanup(restore)
	AppVersion, Commit, BuildTime = "captured_version", "captured_commit", "captured_time"
	authority, err := defaultAIRuntimeToolAuthority()
	restore() // Restore immediately, including on a constructor failure.
	if err != nil {
		t.Fatal(err)
	}
	if runtimeToolOutput(t, authority) != `{"version":"captured_version","commit":"captured_commit","build_time":"captured_time"}` {
		t.Fatal("handler read globals after construction")
	}
	current, err := defaultAIRuntimeToolAuthority()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := json.Marshal(aiRuntimeMetadata{Version: oldVersion, Commit: oldCommit, BuildTime: oldBuildTime})
	if err != nil || runtimeToolOutput(t, current) != string(expected) {
		t.Fatal("fresh authority did not capture restored globals")
	}
}

func TestAIRuntimeToolMetadataValidation(t *testing.T) {
	invalid := []string{string([]byte{0xff}), "private\nvalue", "private\tvalue", "private\x00value", "private\x1fvalue", "private\x7fvalue", "private\u0080value", "private\u009fvalue", strings.Repeat("x", 1025), strings.Repeat("é", 513)}
	for field := range 3 {
		for _, value := range invalid {
			metadata := aiRuntimeMetadata{Version: "private_version", Commit: "private_commit", BuildTime: "private_time"}
			switch field {
			case 0:
				metadata.Version = value
			case 1:
				metadata.Commit = value
			case 2:
				metadata.BuildTime = value
			}
			authority, err := newAIRuntimeToolAuthority(metadata)
			if authority != nil || err != tool.ErrInvalidExecutor || err.Error() != "ai/tool: invalid executor" {
				t.Fatal("invalid metadata not safely rejected")
			}
		}
	}
	for _, value := range []string{"", "release \"quoted\" \\ path", "版本 é", strings.Repeat("&", 1024), strings.Repeat("é", 512)} {
		metadata := aiRuntimeMetadata{Version: value, Commit: value, BuildTime: value}
		authority, err := newAIRuntimeToolAuthority(metadata)
		if err != nil {
			t.Fatal("valid metadata rejected", err)
		}
		output := runtimeToolOutput(t, authority)
		var decoded aiRuntimeMetadata
		if !utf8.ValidString(output) || len(output) > 20*1024 || json.Unmarshal([]byte(output), &decoded) != nil || decoded != metadata {
			t.Fatal("output truncated, normalized or unbounded")
		}
	}
}
