package tool

import "errors"

// ErrInvalidCatalog reports invalid caller configuration without reflecting it.
var ErrInvalidCatalog = errors.New("ai/tool: invalid catalog")

// String reports only fixed admission classifications, never names or arguments.
func (a Admission) String() string {
	if a.status == Admitted && a.reason == Allowed {
		return "tool admission: ADMITTED / ALLOWED"
	}
	switch a.reason {
	case MalformedCall:
		return "tool admission: REJECTED / MALFORMED_CALL"
	case UnknownTool:
		return "tool admission: REJECTED / UNKNOWN_TOOL"
	case InvalidArguments:
		return "tool admission: REJECTED / INVALID_ARGUMENTS"
	case PolicyRejected:
		return "tool admission: REJECTED / POLICY_REJECTED"
	default:
		return "tool admission: REJECTED / NOT_EVALUATED"
	}
}

// GoString keeps Go-syntax diagnostic formatting payload-free as well.
func (a Admission) GoString() string { return a.String() }
