package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
	"github.com/spf13/cobra"
)

const aiCanary = "dummy-AI-CREDENTIAL-CANARY"

type aiFakeProvider func(context.Context, ai.Request) (ai.Result, error)

func (f aiFakeProvider) Execute(ctx context.Context, r ai.Request) (ai.Result, error) {
	return f(ctx, r)
}
func aiArgs() []string {
	return []string{"ai", "prompt", "--provider", "openai", "--model", "gpt-4.1-mini-2025-04-14", "--text", "  Dummy note.\n", "--allow-network"}
}
func aiTestRoot(deps aiDependencies) *cobra.Command {
	root := &cobra.Command{Use: "forge", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(newAICmd(deps))
	return root
}
func aiTestExecute(t *testing.T, deps aiDependencies, args []string, ctx context.Context, w io.Writer) (error, string) {
	t.Helper()
	root := aiTestRoot(deps)
	var stderr bytes.Buffer
	root.SetOut(w)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	// cmd/forge prints the returned error; inspect the same representation.
	if err != nil {
		fmt.Fprint(&stderr, err)
	}
	return err, stderr.String()
}
func aiCheckSecret(t *testing.T, err error, streams ...string) {
	t.Helper()
	for _, s := range streams {
		if strings.Contains(s, aiCanary) {
			t.Fatal("secret in output")
		}
	}
	if err == nil {
		return
	}
	if strings.Contains(fmt.Sprintf("%+v %#v", err, err), aiCanary) {
		t.Fatal("secret error")
	}
	if multi, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range multi.Unwrap() {
			aiCheckSecret(t, e)
		}
	}
	if one, ok := err.(interface{ Unwrap() error }); ok {
		aiCheckSecret(t, one.Unwrap())
	}
}
func TestAIPromptSuccessDefaultsAndExplicitBounds(t *testing.T) {
	for _, suffix := range [][]string{nil, {"--timeout", "120s", "--max-output-tokens", "2048"}, {"--timeout", "1s", "--max-output-tokens", "16"}} {
		lookups, factories, calls := 0, 0, 0
		var got ai.Request
		var deadline time.Duration
		deps := aiDependencies{
			lookupKey: func() string { lookups++; return aiCanary },
			newProvider: func(key string) (ai.Provider, error) {
				factories++
				if key != aiCanary {
					t.Fatal("wrong key")
				}
				return aiFakeProvider(func(ctx context.Context, r ai.Request) (ai.Result, error) {
					calls++
					got = r
					end, ok := ctx.Deadline()
					if !ok {
						t.Fatal("missing deadline")
					}
					deadline = time.Until(end)
					return ai.Result{Text: "Demo Friday.", Model: r.Model}, nil
				}), nil
			},
		}
		var out bytes.Buffer
		err, stderr := aiTestExecute(t, deps, append(aiArgs(), suffix...), context.Background(), &out)
		if err != nil || stderr != "" || out.String() != "Demo Friday.\n" || lookups != 1 || factories != 1 || calls != 1 || ExitCode(err) != 0 {
			t.Fatal("success contract", err)
		}
		wantTokens := 1024
		wantTimeout := 30 * time.Second
		if len(suffix) > 0 {
			if suffix[1] == "120s" {
				wantTokens = 2048
				wantTimeout = 120 * time.Second
			} else {
				wantTokens = 16
				wantTimeout = time.Second
			}
		}
		if got.Text != "  Dummy note.\n" || got.Model != "gpt-4.1-mini-2025-04-14" || got.MaxOutputTokens != wantTokens || deadline <= 0 || deadline > wantTimeout {
			t.Fatal("defaults/preservation")
		}
		aiCheckSecret(t, err, out.String(), stderr)
	}
}
func TestAIInvalidInputBeforeCredentialOrFactory(t *testing.T) {
	base := aiArgs()
	cases := map[string][]string{
		"missing all":      {"ai", "prompt"},
		"missing provider": {"ai", "prompt", "--model", "m", "--text", "dummy", "--allow-network"},
		"missing model":    {"ai", "prompt", "--provider", "openai", "--text", "dummy", "--allow-network"},
		"missing text":     {"ai", "prompt", "--provider", "openai", "--model", "m", "--allow-network"},
		"missing allow":    base[:len(base)-1],
		"false allow":      append(append([]string{}, base...), "--allow-network=false"),
	}
	for name, suffix := range map[string][]string{
		"provider":         {"--provider", "OPENAI"},
		"model":            {"--model", "bad secret model"},
		"empty text":       {"--text", ""},
		"whitespace":       {"--text", " \t\n"},
		"invalid UTF8":     {"--text", "\xff"},
		"oversized":        {"--text", strings.Repeat("a", ai.MaxInputBytes+1)},
		"tokens zero":      {"--max-output-tokens", "0"},
		"tokens negative":  {"--max-output-tokens", "-1"},
		"tokens small":     {"--max-output-tokens", "15"},
		"tokens large":     {"--max-output-tokens", "2049"},
		"timeout zero":     {"--timeout", "0"},
		"timeout negative": {"--timeout", "-1s"},
		"timeout large":    {"--timeout", "121s"},
		"positional":       {"unexpected prompt"},
		"bad duration":     {"--timeout", aiCanary},
		"bad int":          {"--max-output-tokens", aiCanary},
		"bad bool":         {"--allow-network=" + aiCanary},
		"api key":          {"--api-key", aiCanary},
		"endpoint":         {"--endpoint", "https://example.invalid"},
		"json":             {"--json"}, "stream": {"--stream"}, "file": {"--file", "file.txt"},
	} {
		cases[name] = append(append([]string{}, base...), suffix...)
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			deps := aiDependencies{
				lookupKey:   func() string { t.Fatal("credential read for invalid request"); return "" },
				newProvider: func(string) (ai.Provider, error) { t.Fatal("provider constructed"); return nil, nil },
			}
			var out bytes.Buffer
			err, stderr := aiTestExecute(t, deps, args, context.Background(), &out)
			wantCategory := ai.ErrInvalidRequest
			if name == "missing allow" || name == "false allow" {
				wantCategory = ai.ErrAuthorization
			}
			if !errors.Is(err, wantCategory) || out.Len() != 0 || ExitCode(err) != 1 {
				t.Fatal("invalid invocation accepted")
			}
			if strings.Contains(stderr, "unexpected prompt") || strings.Contains(stderr, "bad secret model") {
				t.Fatal("input leaked")
			}
			aiCheckSecret(t, err, out.String(), stderr)
		})
	}
}
func TestAICredentialFailuresDoNotCallProvider(t *testing.T) {
	for _, key := range []string{"", " ", aiCanary + "\n", strings.Repeat("a", 16*1024+1)} {
		reads := 0
		deps := aiDependencies{lookupKey: func() string { reads++; return key }, newProvider: func(string) (ai.Provider, error) { t.Fatal("invalid key reached factory"); return nil, nil }}
		var out bytes.Buffer
		err, stderr := aiTestExecute(t, deps, aiArgs(), context.Background(), &out)
		if !errors.Is(err, ai.ErrAuthentication) || reads != 1 || out.Len() != 0 {
			t.Fatal("credential contract", err)
		}
		aiCheckSecret(t, err, stderr)
	}
}

type aiBadWriter struct{ short bool }

func (w aiBadWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) - 1, nil
	}
	return 0, errors.New(aiCanary)
}
func TestAIOutputAndFailures(t *testing.T) {
	cases := []struct {
		name, text string
		failure    error
		want       error
		exit       int
	}{
		{"newline", "OK\n", nil, nil, 0},
		{"command", "rm -rf /; $(echo data)", nil, nil, 0},
		{"ansi", "\x1b[31mred", nil, ai.ErrMalformedResponse, 1},
		{"C1", "\u009b31mred", nil, ai.ErrMalformedResponse, 1},
		{"secret", aiCanary, nil, ai.ErrMalformedResponse, 1},
		{"refused", "partial", fmt.Errorf("%s: %w", aiCanary, ai.ErrRefused), ai.ErrRefused, 1},
		{"authentication", "partial", fmt.Errorf("%s: %w", aiCanary, ai.ErrAuthentication), ai.ErrAuthentication, 1},
		{"rate", "partial", fmt.Errorf("%s: %w", aiCanary, ai.ErrRateLimited), ai.ErrRateLimited, 1},
		{"cancel", "partial", fmt.Errorf("%s: %w", aiCanary, context.Canceled), context.Canceled, 130},
		{"deadline", "partial", fmt.Errorf("%s: %w", aiCanary, context.DeadlineExceeded), context.DeadlineExceeded, 1},
		{"unknown", "partial", errors.New(aiCanary), ai.ErrProvider, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			deps := aiDependencies{lookupKey: func() string { return aiCanary }, newProvider: func(string) (ai.Provider, error) {
				return aiFakeProvider(func(context.Context, ai.Request) (ai.Result, error) {
					calls++
					return ai.Result{Text: tc.text, Model: "m"}, tc.failure
				}), nil
			}}
			var out bytes.Buffer
			err, stderr := aiTestExecute(t, deps, aiArgs(), context.Background(), &out)
			if ExitCode(err) != tc.exit || !errors.Is(err, tc.want) || calls != 1 {
				t.Fatal("result category", err)
			}
			if err != nil && out.Len() != 0 {
				t.Fatal("partial output")
			}
			if err == nil {
				want := tc.text
				if !strings.HasSuffix(want, "\n") {
					want += "\n"
				}
				if out.String() != want || stderr != "" {
					t.Fatal("text output")
				}
			}
			aiCheckSecret(t, err, out.String(), stderr)
		})
	}
	deps := aiDependencies{lookupKey: func() string { return aiCanary }, newProvider: func(string) (ai.Provider, error) {
		return aiFakeProvider(func(context.Context, ai.Request) (ai.Result, error) { return ai.Result{Text: "OK", Model: "m"}, nil }), nil
	}}
	for _, short := range []bool{false, true} {
		err, stderr := aiTestExecute(t, deps, aiArgs(), context.Background(), aiBadWriter{short: short})
		if err == nil || ExitCode(err) != 1 {
			t.Fatal("writer failure accepted")
		}
		aiCheckSecret(t, err, stderr)
	}
}
func TestAIContextPropagation(t *testing.T) {
	for _, mode := range []string{"pre-cancel", "cancel", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if mode == "pre-cancel" {
					cancel()
				}
				entered := make(chan struct{})
				calls := 0
				deps := aiDependencies{
					lookupKey: func() string {
						if mode == "pre-cancel" {
							t.Fatal("key read after cancel")
						}
						return aiCanary
					},
					newProvider: func(string) (ai.Provider, error) {
						return aiFakeProvider(func(ctx context.Context, r ai.Request) (ai.Result, error) {
							calls++
							close(entered)
							<-ctx.Done()
							return ai.Result{}, ctx.Err()
						}), nil
					},
				}
				if mode == "cancel" {
					go func() { <-entered; cancel() }()
				}
				var out bytes.Buffer
				err, stderr := aiTestExecute(t, deps, aiArgs(), ctx, &out)
				want := context.Canceled
				exit := 130
				wantCalls := 1
				if mode == "timeout" {
					want = context.DeadlineExceeded
					exit = 1
				}
				if mode == "pre-cancel" {
					wantCalls = 0
				}
				if !errors.Is(err, want) || ExitCode(err) != exit || calls != wantCalls || out.Len() != 0 {
					t.Fatal("context semantics", err)
				}
				aiCheckSecret(t, err, stderr)
			})
		})
	}
}
func TestAIHelpDoesNotReadCredential(t *testing.T) {
	deps := aiDependencies{lookupKey: func() string { t.Fatal("help read credential"); return "" }, newProvider: func(string) (ai.Provider, error) { t.Fatal("help constructed provider"); return nil, nil }}
	for _, args := range [][]string{{"ai", "--help"}, {"ai", "prompt", "--help"}} {
		var out bytes.Buffer
		err, _ := aiTestExecute(t, deps, args, context.Background(), &out)
		if err != nil {
			t.Fatal(err)
		}
		if len(args) == 3 {
			for _, flag := range []string{"--provider", "--model", "--text", "--allow-network", "--allow-tools", "--timeout", "--max-output-tokens"} {
				if !strings.Contains(out.String(), flag) {
					t.Fatal("missing help flag")
				}
			}
		}
	}
}
func TestAINoExecutionOrFilesystemDiscovery(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "must-not-exist")
	output := "New-Item -Path '" + marker + "'; rm -rf /"
	deps := aiDependencies{lookupKey: func() string { return aiCanary }, newProvider: func(string) (ai.Provider, error) {
		return aiFakeProvider(func(_ context.Context, r ai.Request) (ai.Result, error) {
			if r.Text != "  Dummy note.\n" {
				t.Fatal("unexpected discovery input")
			}
			return ai.Result{Text: output, Model: r.Model}, nil
		}), nil
	}}
	var out bytes.Buffer
	err, _ := aiTestExecute(t, deps, aiArgs(), context.Background(), &out)
	if err != nil || out.String() != output+"\n" {
		t.Fatal("command-like text changed")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("output executed or artifacts created")
	}
}

func TestAIFactoryErrorsAndMixedCancellationAreSafe(t *testing.T) {
	for _, factoryError := range []error{errors.New(aiCanary), fmt.Errorf("%s: %w", aiCanary, ai.ErrAuthentication)} {
		deps := aiDependencies{lookupKey: func() string { return aiCanary }, newProvider: func(string) (ai.Provider, error) { return nil, factoryError }}
		var out bytes.Buffer
		err, stderr := aiTestExecute(t, deps, aiArgs(), context.Background(), &out)
		if err == nil || out.Len() != 0 || ExitCode(err) != 1 {
			t.Fatal("factory failure accepted")
		}
		aiCheckSecret(t, err, out.String(), stderr)
	}
	deps := aiDependencies{lookupKey: func() string { return aiCanary }, newProvider: func(string) (ai.Provider, error) {
		return aiFakeProvider(func(context.Context, ai.Request) (ai.Result, error) {
			return ai.Result{Text: "partial", Model: "m"}, errors.Join(context.Canceled, fmt.Errorf("%s: %w", aiCanary, ai.ErrTransport))
		}), nil
	}}
	var out bytes.Buffer
	err, stderr := aiTestExecute(t, deps, aiArgs(), context.Background(), &out)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, ai.ErrTransport) || ExitCode(err) != 1 || out.Len() != 0 {
		t.Fatal("mixed failure became clean cancel")
	}
	aiCheckSecret(t, err, stderr)
}

func TestAIMixedUnknownFailureExitCode(t *testing.T) {
	unknown := errors.New(aiCanary)
	for _, tc := range []struct {
		name     string
		failure  error
		category error
		wantExit int
	}{
		{"pure cancellation", context.Canceled, nil, 130},
		{"wrapped cancellation", fmt.Errorf("%s: %w", aiCanary, context.Canceled), nil, 130},
		{"typed mixed", errors.Join(context.Canceled, ai.ErrTransport), ai.ErrTransport, 1},
		{"unknown mixed", errors.Join(context.Canceled, unknown), ai.ErrProvider, 1},
		{"reversed mixed", errors.Join(unknown, context.Canceled), ai.ErrProvider, 1},
		{"wrapped mixed", fmt.Errorf("%s: %w", aiCanary, errors.Join(context.Canceled, unknown)), ai.ErrProvider, 1},
		{"nested mixed", errors.Join(context.Canceled, errors.Join(unknown)), ai.ErrProvider, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			deps := aiDependencies{
				lookupKey: func() string { return aiCanary },
				newProvider: func(string) (ai.Provider, error) {
					return aiFakeProvider(func(context.Context, ai.Request) (ai.Result, error) {
						calls++
						return ai.Result{Text: "partial", Model: "m"}, tc.failure
					}), nil
				},
			}
			var out bytes.Buffer
			err, stderr := aiTestExecute(t, deps, aiArgs(), context.Background(), &out)
			if got := ExitCode(err); got != tc.wantExit {
				t.Errorf("ExitCode = %d; want %d", got, tc.wantExit)
			}
			if !errors.Is(err, context.Canceled) {
				t.Error("cancellation category lost")
			}
			if tc.category != nil && !errors.Is(err, tc.category) {
				t.Error("mixed failure category lost")
			}
			if out.Len() != 0 || calls != 1 {
				t.Error("partial stdout or incorrect provider call count")
			}
			if errors.Is(err, unknown) {
				t.Error("raw cause retained")
			}
			aiCheckSecret(t, err, out.String(), stderr)
			if ExitCode(ai.SafeError(err)) != tc.wantExit {
				t.Error("repeated sanitization changed exit status")
			}
		})
	}
}

type aiFakeToolProvider func(context.Context, ai.Request, *tool.Authority) (ai.Result, error)

func (f aiFakeToolProvider) ExecuteAuthorizedFunctionRoundTrip(ctx context.Context, request ai.Request, authority *tool.Authority) (ai.Result, error) {
	return f(ctx, request, authority)
}

type aiToolFixture struct {
	deps                 aiDependencies
	order                []string
	textCalls, toolCalls int
	ctx                  context.Context
	request              ai.Request
	authority            *tool.Authority
}

func newAIToolFixture(t *testing.T) *aiToolFixture {
	t.Helper()
	f := &aiToolFixture{}
	f.deps = aiDependencies{
		lookupKey: func() string { f.order = append(f.order, "key"); return aiCanary },
		newProvider: func(key string) (ai.Provider, error) {
			f.order = append(f.order, "text factory")
			if key != aiCanary {
				t.Fatal("wrong text key")
			}
			return aiFakeProvider(func(_ context.Context, request ai.Request) (ai.Result, error) {
				f.textCalls++
				return ai.Result{Text: "provider final text", Model: request.Model}, nil
			}), nil
		},
		newToolAuthority: func() (*tool.Authority, error) {
			f.order = append(f.order, "authority")
			return defaultAIRuntimeToolAuthority()
		},
		newToolProvider: func(key string) (aiToolProvider, error) {
			f.order = append(f.order, "tool factory")
			if key != aiCanary {
				t.Fatal("wrong tool key")
			}
			return aiFakeToolProvider(func(ctx context.Context, request ai.Request, authority *tool.Authority) (ai.Result, error) {
				f.toolCalls++
				f.ctx, f.request, f.authority = ctx, request, authority
				return ai.Result{Text: "provider final text", Model: request.Model}, nil
			}), nil
		},
	}
	return f
}

func aiToolArgs() []string { return append(aiArgs(), "--allow-tools") }

func TestAIToolOptInAndDefaultOff(t *testing.T) {
	for _, mode := range []string{"default", "false", "enabled", "broken tool hooks"} {
		t.Run(mode, func(t *testing.T) {
			f := newAIToolFixture(t)
			args := aiArgs()
			wantOrder, textCalls, toolCalls := []string{"key", "text factory"}, 1, 0
			switch mode {
			case "false":
				args = append(args, "--allow-tools=false")
			case "broken tool hooks":
				f.deps.newToolAuthority, f.deps.newToolProvider = nil, nil
			case "enabled":
				args = aiToolArgs()
				f.deps.newProvider = nil // Tool mode does not require text factory.
				wantOrder, textCalls, toolCalls = []string{"authority", "key", "tool factory"}, 0, 1
			}
			var out bytes.Buffer
			err, stderr := aiTestExecute(t, f.deps, args, context.Background(), &out)
			if err != nil || stderr != "" || out.String() != "provider final text\n" || ExitCode(err) != 0 || !reflect.DeepEqual(f.order, wantOrder) || f.textCalls != textCalls || f.toolCalls != toolCalls {
				t.Fatal("opt-in wiring/counts", err)
			}
			if toolCalls == 1 {
				defs := f.authority.Definitions()
				if len(defs) != 1 || defs[0].Name != "forge_runtime_info" || len(defs[0].Parameters) != 0 || f.authority.Executor() == nil {
					t.Fatal("silent capability expansion")
				}
				if f.request != (ai.Request{Text: "  Dummy note.\n", Model: "gpt-4.1-mini-2025-04-14", MaxOutputTokens: 1024}) {
					t.Fatal("tool request rewritten")
				}
				if f.ctx.Err() != context.Canceled {
					t.Fatal("operation context not released")
				}
			}
		})
	}
}

func TestAIToolInvalidInputBeforeDependencies(t *testing.T) {
	for name, suffix := range map[string][]string{
		"network missing": {"--allow-network=false"},
		"tool bool":       {"--allow-tools=" + aiCanary},
		"provider":        {"--provider", aiCanary},
		"model":           {"--model", "bad model"},
		"text":            {"--text", ""},
		"timeout":         {"--timeout", "121s"},
		"tokens":          {"--max-output-tokens", "0"},
		"positional":      {aiCanary},
	} {
		t.Run(name, func(t *testing.T) {
			f := newAIToolFixture(t)
			var out bytes.Buffer
			err, stderr := aiTestExecute(t, f.deps, append(aiToolArgs(), suffix...), context.Background(), &out)
			want := ai.ErrInvalidRequest
			if name == "network missing" {
				want = ai.ErrAuthorization
			}
			if !errors.Is(err, want) || out.Len() != 0 || len(f.order) != 0 || f.toolCalls != 0 || f.textCalls != 0 {
				t.Fatal("invalid input reached dependencies", err)
			}
			aiCheckSecret(t, err, stderr)
		})
	}
	f := newAIToolFixture(t)
	args := append(aiArgs()[:len(aiArgs())-1], "--allow-tools")
	var out bytes.Buffer
	err, stderr := aiTestExecute(t, f.deps, args, context.Background(), &out)
	if !errors.Is(err, ai.ErrAuthorization) || len(f.order) != 0 || out.Len() != 0 {
		t.Fatal("tools bypassed network consent")
	}
	aiCheckSecret(t, err, stderr)
}

func TestAIToolAuthorityAndFactoryFailures(t *testing.T) {
	for _, mode := range []string{"nil key hook", "nil authority hook", "nil tool hook", "authority error", "nil authority", "zero authority", "empty authority", "invalid key", "factory error", "nil provider", "typed nil provider"} {
		t.Run(mode, func(t *testing.T) {
			f := newAIToolFixture(t)
			want, order := ai.ErrInvalidRequest, []string(nil)
			switch mode {
			case "nil key hook":
				f.deps.lookupKey = nil
			case "nil authority hook":
				f.deps.newToolAuthority = nil
			case "nil tool hook":
				f.deps.newToolProvider = nil
			case "authority error", "nil authority", "zero authority", "empty authority":
				order = []string{"authority"}
				f.deps.newToolAuthority = func() (*tool.Authority, error) {
					f.order = append(f.order, "authority")
					switch mode {
					case "authority error":
						return nil, fmt.Errorf("%s: %w", aiCanary, tool.ErrInvalidExecutor)
					case "zero authority":
						return &tool.Authority{}, nil
					case "empty authority":
						return tool.NewAuthority(nil)
					default:
						return nil, nil
					}
				}
				if mode == "authority error" {
					want = ai.ErrProvider
				}
			case "invalid key":
				order = []string{"authority", "key"}
				want = ai.ErrAuthentication
				f.deps.lookupKey = func() string { f.order = append(f.order, "key"); return aiCanary + "\n" }
			case "factory error", "nil provider", "typed nil provider":
				order = []string{"authority", "key", "tool factory"}
				f.deps.newToolProvider = func(string) (aiToolProvider, error) {
					f.order = append(f.order, "tool factory")
					if mode == "factory error" {
						return nil, errors.New(aiCanary)
					}
					if mode == "typed nil provider" {
						return aiFakeToolProvider(nil), nil
					}
					return nil, nil
				}
				if mode == "factory error" {
					want = ai.ErrProvider
				}
			}
			var out bytes.Buffer
			err, stderr := aiTestExecute(t, f.deps, aiToolArgs(), context.Background(), &out)
			if !errors.Is(err, want) || !reflect.DeepEqual(f.order, order) || f.toolCalls != 0 || f.textCalls != 0 || out.Len() != 0 || ExitCode(err) != 1 {
				t.Fatal("dependency failure order/classification", err)
			}
			aiCheckSecret(t, err, stderr)
		})
	}
}

func TestAIToolOutputAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name, text, model string
		failure, want     error
		exit              int
	}{
		{"plain", "answer", "m", nil, nil, 0},
		{"newline", "answer\n", "m", nil, nil, 0},
		{"text reflection", aiCanary, "m", nil, ai.ErrMalformedResponse, 1},
		{"model reflection", "answer", aiCanary, nil, ai.ErrMalformedResponse, 1},
		{"invalid text", "\x1b[31m", "m", nil, ai.ErrMalformedResponse, 1},
		{"authorization", "partial", "m", fmt.Errorf("%s: %w", aiCanary, ai.ErrAuthorization), ai.ErrAuthorization, 1},
		{"execution denied", "partial", "m", tool.ErrExecutionDenied, ai.ErrProvider, 1},
		{"handler failed", "partial", "m", tool.ErrHandlerFailed, ai.ErrProvider, 1},
		{"provider", "partial", "m", fmt.Errorf("%s: %w", aiCanary, ai.ErrTransport), ai.ErrTransport, 1},
		{"canceled", "partial", "m", fmt.Errorf("%s: %w", aiCanary, context.Canceled), context.Canceled, 130},
		{"deadline", "partial", "m", context.DeadlineExceeded, context.DeadlineExceeded, 1},
		{"unknown", "partial", "m", errors.New(aiCanary), ai.ErrProvider, 1},
		{"mixed", "partial", "m", errors.Join(context.Canceled, errors.New(aiCanary)), ai.ErrProvider, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAIToolFixture(t)
			f.deps.newToolProvider = func(string) (aiToolProvider, error) {
				return aiFakeToolProvider(func(context.Context, ai.Request, *tool.Authority) (ai.Result, error) {
					f.toolCalls++
					return ai.Result{Text: tc.text, Model: tc.model, Usage: &ai.Usage{InputTokens: 20, OutputTokens: 2048}}, tc.failure
				}), nil
			}
			var out bytes.Buffer
			err, stderr := aiTestExecute(t, f.deps, aiToolArgs(), context.Background(), &out)
			if !errors.Is(err, tc.want) || ExitCode(err) != tc.exit || f.toolCalls != 1 {
				t.Fatal("tool error category", err)
			}
			if err != nil {
				if out.Len() != 0 {
					t.Fatal("partial stdout")
				}
			} else {
				want := tc.text
				if !strings.HasSuffix(want, "\n") {
					want += "\n"
				}
				if out.String() != want || stderr != "" {
					t.Fatal("stdout not final text only")
				}
			}
			aiCheckSecret(t, err, out.String(), stderr)
		})
	}
	for _, short := range []bool{false, true} {
		f := newAIToolFixture(t)
		err, stderr := aiTestExecute(t, f.deps, aiToolArgs(), context.Background(), aiBadWriter{short: short})
		if !errors.Is(err, io.ErrShortWrite) || ExitCode(err) != 1 || f.toolCalls != 1 {
			t.Fatal("writer error not redacted")
		}
		aiCheckSecret(t, err, stderr)
	}
}

func TestAIToolTimeoutAndCancellation(t *testing.T) {
	for _, budget := range []time.Duration{time.Second, 120 * time.Second} {
		synctest.Test(t, func(t *testing.T) {
			f := newAIToolFixture(t)
			start := time.Now()
			f.deps.newToolProvider = func(string) (aiToolProvider, error) {
				return aiFakeToolProvider(func(ctx context.Context, _ ai.Request, _ *tool.Authority) (ai.Result, error) {
					f.toolCalls++
					deadline, ok := ctx.Deadline()
					if !ok || deadline.After(start.Add(budget)) {
						t.Fatal("CLI timeout lost")
					}
					<-ctx.Done()
					// Even a provider ignoring cancellation on return cannot print output.
					return ai.Result{Text: "late output", Model: "m"}, nil
				}), nil
			}
			var out bytes.Buffer
			err, _ := aiTestExecute(t, f.deps, append(aiToolArgs(), "--timeout", budget.String()), context.Background(), &out)
			if !errors.Is(err, context.DeadlineExceeded) || out.Len() != 0 || f.toolCalls != 1 || time.Since(start) != budget {
				t.Fatal("CLI whole-call deadline", err)
			}
		})
	}
	f := newAIToolFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	err, _ := aiTestExecute(t, f.deps, aiToolArgs(), ctx, &out)
	if !errors.Is(err, context.Canceled) || len(f.order) != 0 || f.toolCalls != 0 || out.Len() != 0 {
		t.Fatal("pre-cancel reached dependencies")
	}
}

func TestAIToolHelpAndDefaultDependencies(t *testing.T) {
	f := newAIToolFixture(t)
	var out bytes.Buffer
	err, _ := aiTestExecute(t, f.deps, []string{"ai", "prompt", "--help"}, context.Background(), &out)
	if err != nil || len(f.order) != 0 {
		t.Fatal("help used dependencies")
	}
	for _, text := range []string{"--allow-tools", "text-only", "forge_runtime_info", "two provider requests", "one read-only tool execution", "No filesystem, subprocess, or external-network tool handlers", "shell history"} {
		if !strings.Contains(out.String(), text) {
			t.Fatal("missing bounded help contract")
		}
	}
	root := aiTestRoot(f.deps)
	command, _, err := root.Find([]string{"ai", "prompt"})
	if err != nil || command.Flags().Lookup("allow-tools").DefValue != "false" {
		t.Fatal("tools default enabled")
	}
	deps := defaultAIDependencies()
	if deps.lookupKey == nil || deps.newProvider == nil || deps.newToolProvider == nil || deps.newToolAuthority == nil {
		t.Fatal("default hooks missing")
	}
	authority, err := deps.newToolAuthority()
	if err != nil {
		t.Fatal(err)
	}
	defs := authority.Definitions()
	if len(defs) != 1 || defs[0].Name != "forge_runtime_info" || len(defs[0].Parameters) != 0 {
		t.Fatal("default capability expanded")
	}
	// Constructor-only: validate default tool wiring without Execute or key lookup.
	p, err := deps.newToolProvider(aiCanary)
	if err != nil || nilAIToolProvider(p) {
		t.Fatal("default tool provider unavailable")
	}
}
