package agent

// State is a fixed operation classification, never execution authority.
type State uint8

const (
	StateUnknown State = iota
	StateReady
	StateRunning
	StateSucceeded
	StateFailed
	StateCanceled
)

func (s State) String() string {
	switch s {
	case StateReady:
		return "ready"
	case StateRunning:
		return "running"
	case StateSucceeded:
		return "succeeded"
	case StateFailed:
		return "failed"
	case StateCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}
