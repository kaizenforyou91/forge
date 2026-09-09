package cli

import (
	"context"
	"encoding/json"
	"unicode/utf8"

	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

const maxAIRuntimeMetadataBytes = 1024

type aiRuntimeMetadata struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

// newAIRuntimeToolAuthority captures read-only metadata as one built-in C5
// binding. forge_runtime_info availability does not mean model authorization:
// C6 configuration still needs B2 declaration, Stage A admission and C1
// execution authority. Construction does not expose a CLI command or flag.
func newAIRuntimeToolAuthority(metadata aiRuntimeMetadata) (*tool.Authority, error) {
	for _, field := range []string{metadata.Version, metadata.Commit, metadata.BuildTime} {
		if !utf8.ValidString(field) || len(field) > maxAIRuntimeMetadataBytes {
			return nil, tool.ErrInvalidExecutor
		}
		for _, r := range field {
			if r < 0x20 || r >= 0x7f && r <= 0x9f {
				return nil, tool.ErrInvalidExecutor
			}
		}
	}
	// Typed JSON fixes field order and owns the snapshot before publication.
	// Even six-byte JSON escaping of every input byte stays below 20 KiB,
	// comfortably within C1's 64 KiB output contract.
	body, err := json.Marshal(metadata)
	if err != nil || !utf8.Valid(body) || len(body) > 20*1024 {
		return nil, tool.ErrInvalidExecutor
	}
	output := string(body)
	return tool.NewAuthority([]tool.Binding{{
		Definition: tool.Definition{Name: "forge_runtime_info", Parameters: nil},
		Handler: func(ctx context.Context, _ tool.Call) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			return output, nil
		},
	}})
}

// defaultAIRuntimeToolAuthority reads only the existing version-command
// globals at construction. The published handler never accesses them again.
// Build metadata must not be mutated concurrently with this snapshot read.
func defaultAIRuntimeToolAuthority() (*tool.Authority, error) {
	return newAIRuntimeToolAuthority(aiRuntimeMetadata{
		Version: AppVersion, Commit: Commit, BuildTime: BuildTime,
	})
}
