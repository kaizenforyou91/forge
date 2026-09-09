package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/kaizenforyou91/forge/internal/aiprovider/openai"
	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
	"github.com/spf13/cobra"
)

// Tool execution is CLI-local composition, not an extension of ai.Provider.
type aiToolProvider interface {
	ExecuteAuthorizedFunctionRoundTrip(context.Context, ai.Request, *tool.Authority) (ai.Result, error)
}

// Dependencies belong to one command tree, never mutable global hooks.
type aiDependencies struct {
	lookupKey        func() string
	newProvider      func(string) (ai.Provider, error)
	newToolProvider  func(string) (aiToolProvider, error)
	newToolAuthority func() (*tool.Authority, error)
}

func defaultAIDependencies() aiDependencies {
	return aiDependencies{
		lookupKey:        func() string { return os.Getenv("FORGE_OPENAI_API_KEY") },
		newProvider:      func(key string) (ai.Provider, error) { return openai.New(key) },
		newToolProvider:  func(key string) (aiToolProvider, error) { return openai.New(key) },
		newToolAuthority: defaultAIRuntimeToolAuthority,
	}
}
func newAICmd(deps aiDependencies) *cobra.Command {
	cmd := &cobra.Command{Use: "ai", Short: "AI text operations (development feature)", Args: cobra.NoArgs}
	var provider, model, text string
	var allowNetwork, allowTools bool
	var timeout time.Duration
	var tokens int
	prompt := &cobra.Command{
		Use:   "prompt",
		Short: "Send one explicit text prompt to OpenAI",
		Long: "Send one explicit text prompt to OpenAI and print only completed text.\n" +
			"Requires --allow-network to permit requests to OpenAI and process-scoped FORGE_OPENAI_API_KEY.\n" +
			"Without --allow-tools, this remains a text-only prompt.\n" +
			"With --allow-tools, Forge may expose only its built-in read-only forge_runtime_info metadata tool for this invocation.\n" +
			"Tool mode permits at most two provider requests and one read-only tool execution.\n" +
			"No filesystem, subprocess, or external-network tool handlers are available.\n" +
			"Use dummy/non-sensitive text: --text may appear in shell history and process arguments.\n" +
			"No file discovery, streaming, retries, or recursive tool loop.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return ai.ErrInvalidRequest
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			req := ai.Request{Text: text, Model: model, MaxOutputTokens: tokens}
			if err := req.Validate(); err != nil {
				return err
			}
			if err := ai.ValidateTimeout(timeout); err != nil {
				return err
			}
			if provider != "openai" {
				return fmt.Errorf("%w: provider must be openai", ai.ErrInvalidRequest)
			}
			if !allowNetwork {
				return fmt.Errorf("%w: --allow-network must be true", ai.ErrAuthorization)
			}
			if err := cmd.Context().Err(); err != nil {
				return err
			}
			if deps.lookupKey == nil {
				return ai.ErrInvalidRequest
			}
			var authority *tool.Authority
			if allowTools {
				if deps.newToolProvider == nil || deps.newToolAuthority == nil {
					return ai.ErrInvalidRequest
				}
				var err error
				authority, err = deps.newToolAuthority()
				if err != nil {
					return ai.SafeError(err)
				}
				if authority.Executor() == nil || len(authority.Definitions()) == 0 {
					return ai.ErrInvalidRequest
				}
			} else if deps.newProvider == nil {
				return ai.ErrInvalidRequest
			}
			key := deps.lookupKey()
			// Validate using the same adapter constructor even with an injected factory.
			// Construction is side-effect-free; key is never persisted or printed.
			if _, err := openai.New(key); err != nil {
				return ai.SafeError(err)
			}
			var result ai.Result
			var err error
			if allowTools {
				p, factoryErr := deps.newToolProvider(key)
				if factoryErr != nil {
					return ai.SafeError(factoryErr)
				}
				if nilAIToolProvider(p) {
					return ai.ErrInvalidRequest
				}
				operationCtx, cancel := context.WithTimeout(cmd.Context(), timeout)
				defer cancel()
				if err := operationCtx.Err(); err != nil {
					return err
				}
				result, err = p.ExecuteAuthorizedFunctionRoundTrip(operationCtx, req, authority)
				if err == nil {
					err = operationCtx.Err()
				}
				if err == nil {
					// C4 usage may aggregate two turns; do not apply a single-turn
					// output-token limit to that total here.
					err = result.Validate()
				}
			} else {
				p, err := deps.newProvider(key)
				if err != nil {
					return ai.SafeError(err)
				}
				executor, err := ai.NewExecutor(p, timeout)
				if err != nil {
					return ai.SafeError(err)
				}
				result, err = executor.Execute(cmd.Context(), req)
				if err != nil {
					return ai.SafeError(err)
				}
			}
			if err != nil {
				return ai.SafeError(err)
			}
			if strings.Contains(result.Text, key) || strings.Contains(result.Model, key) {
				return ai.ErrMalformedResponse
			}
			if err := cmd.Context().Err(); err != nil {
				return err
			}
			output := result.Text
			if !strings.HasSuffix(output, "\n") {
				output += "\n"
			}
			n, err := io.WriteString(cmd.OutOrStdout(), output)
			// Do not expose an arbitrary output writer's error message either.
			if err != nil || n != len(output) {
				return fmt.Errorf("ai: write output: %w", io.ErrShortWrite)
			}
			return nil
		},
	}
	// Cobra/pflag parse errors can contain arbitrary flag values. Keep diagnostics
	// classified and value-free (including invalid duration/int/bool flags).
	prompt.SetFlagErrorFunc(func(*cobra.Command, error) error { return ai.ErrInvalidRequest })
	prompt.Flags().StringVar(&provider, "provider", "", "required provider (openai)")
	prompt.Flags().StringVar(&model, "model", "", "required explicit model ID; no default")
	prompt.Flags().StringVar(&text, "text", "", "required explicit non-sensitive UTF-8 prompt")
	prompt.Flags().BoolVar(&allowNetwork, "allow-network", false, "allow this invocation to send text to OpenAI")
	prompt.Flags().BoolVar(&allowTools, "allow-tools", false, "additionally allow the built-in read-only forge_runtime_info metadata tool")
	prompt.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "request timeout, positive and at most 120s")
	prompt.Flags().IntVar(&tokens, "max-output-tokens", 1024, "output token limit, 16..2048")

	cmd.AddCommand(prompt)
	return cmd
}

// Match the core executor's fail-closed interface-nil handling without changing
// ai.Provider or accepting a typed-nil tool provider from an injected factory.
func nilAIToolProvider(p aiToolProvider) bool {
	if p == nil {
		return true
	}
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
