package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kaizenforyou91/forge/internal/aiprovider/openai"
	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/spf13/cobra"
)

// Dependencies belong to one command tree, never mutable global hooks.
type aiDependencies struct {
	lookupKey   func() string
	newProvider func(string) (ai.Provider, error)
}

func defaultAIDependencies() aiDependencies {
	return aiDependencies{
		lookupKey:   func() string { return os.Getenv("FORGE_OPENAI_API_KEY") },
		newProvider: func(key string) (ai.Provider, error) { return openai.New(key) },
	}
}
func newAICmd(deps aiDependencies) *cobra.Command {
	cmd := &cobra.Command{Use: "ai", Short: "AI text operations (development feature)", Args: cobra.NoArgs}
	var provider, model, text string
	var allowNetwork bool
	var timeout time.Duration
	var tokens int
	prompt := &cobra.Command{
		Use:   "prompt",
		Short: "Send one explicit text prompt to OpenAI",
		Long: "Send one explicit text prompt to OpenAI and print only completed text.\n" +
			"Requires --allow-network and process-scoped FORGE_OPENAI_API_KEY.\n" +
			"Use dummy/non-sensitive text: --text may appear in shell history and process arguments.\n" +
			"No tools, file discovery, streaming, retries, or execution of model output.",
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
			if deps.lookupKey == nil || deps.newProvider == nil {
				return ai.ErrInvalidRequest
			}
			key := deps.lookupKey()
			// Validate using the same adapter constructor even with an injected factory.
			// Construction is side-effect-free; key is never persisted or printed.
			if _, err := openai.New(key); err != nil {
				return ai.SafeError(err)
			}
			p, err := deps.newProvider(key)
			if err != nil {
				return ai.SafeError(err)
			}
			executor, err := ai.NewExecutor(p, timeout)
			if err != nil {
				return ai.SafeError(err)
			}
			result, err := executor.Execute(cmd.Context(), req)
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
	prompt.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "request timeout, positive and at most 120s")
	prompt.Flags().IntVar(&tokens, "max-output-tokens", 1024, "output token limit, 16..2048")

	cmd.AddCommand(prompt)
	return cmd
}
