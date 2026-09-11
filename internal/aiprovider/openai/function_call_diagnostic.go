package openai

import (
	"context"

	"github.com/kaizenforyou91/forge/pkg/ai"
	"github.com/kaizenforyou91/forge/pkg/ai/tool"
)

// FunctionRoundTripStage carries only a bounded internal execution location.
// Its zero value and invalid numeric values render as empty, never dynamic text.
type FunctionRoundTripStage uint8

const (
	StageNone FunctionRoundTripStage = iota
	StageRequestBuild
	StageInitialContext
	StagePost1Request
	StagePost1TextDecode
	StagePost1Validate
	StagePost1Map
	StagePost1Admission
	StagePost1Reflection
	StagePost1Usage
	StageHandlerExecute
	StageContinuationBuild
	StagePost2Request
	StagePost2Decode
	StagePost2Validate
	StageUsageAggregate
	StageFinalContext
	StageComplete
)

func (s FunctionRoundTripStage) String() string {
	switch s {
	case StageRequestBuild:
		return "request_build"
	case StageInitialContext:
		return "initial_context"
	case StagePost1Request:
		return "post1_request"
	case StagePost1TextDecode:
		return "post1_text_decode"
	case StagePost1Validate:
		return "post1_validate"
	case StagePost1Map:
		return "post1_map"
	case StagePost1Admission:
		return "post1_admission"
	case StagePost1Reflection:
		return "post1_reflection"
	case StagePost1Usage:
		return "post1_usage"
	case StageHandlerExecute:
		return "handler_execute"
	case StageContinuationBuild:
		return "continuation_build"
	case StagePost2Request:
		return "post2_request"
	case StagePost2Decode:
		return "post2_decode"
	case StagePost2Validate:
		return "post2_validate"
	case StageUsageAggregate:
		return "usage_aggregate"
	case StageFinalContext:
		return "final_context"
	case StageComplete:
		return "complete"
	default:
		return ""
	}
}

func (s *FunctionRoundTripStage) set(value FunctionRoundTripStage) {
	if s != nil {
		*s = value
	}
}

// ExecuteAuthorizedFunctionRoundTripDiagnostic uses the same C5 authority and
// C4 core as ordinary execution. Only this opt-in entry point returns a stage;
// errors retain the existing safe classification, never raw handler causes.
func (c *Client) ExecuteAuthorizedFunctionRoundTripDiagnostic(ctx context.Context, request ai.Request, authority *tool.Authority) (ai.Result, FunctionRoundTripStage, error) {
	stage := StageRequestBuild
	definitions, executor := authority.Definitions(), authority.Executor()
	if len(definitions) == 0 || executor == nil {
		return ai.Result{}, stage, ai.ErrInvalidRequest
	}
	result, err := c.executeFunctionRoundTrip(ctx, request, definitions, executor, &stage)
	return result, stage, ai.SafeError(err)
}
