// Package tool makes bounded, provider-neutral admission decisions without
// executing tools, performing I/O, or calling caller-supplied code. Admission
// establishes only catalog membership and the documented argument contract;
// it is not execution, safety, success, or persistent authorization.
package tool

// ValueType is an exact JSON type, not a coercion or domain-value constraint.
type ValueType uint8

const (
	InvalidType ValueType = iota
	String
	Number
	Boolean
	Object
	Array
	Null
)

// Parameter describes one root member. Nested containers receive structural
// validation only. Required means present, not nonblank or nonnull.
type Parameter struct {
	Name     string
	Type     ValueType
	Required bool
}

// Definition contains admission data only, never a handler or execution target.
type Definition struct {
	Name       string
	Parameters []Parameter
}

// Call is untrusted data. Arguments must be one strict JSON object. Callers
// must not mutate Arguments concurrently while Admit is reading/copying it.
type Call struct {
	Name      string
	Arguments []byte
}

// Status describes admission only. Its zero value fails closed.
type Status uint8

const (
	Rejected Status = iota
	Admitted
)

// Reason is a payload-free decision classification.
type Reason uint8

const (
	NotEvaluated Reason = iota
	Allowed
	MalformedCall
	UnknownTool
	InvalidArguments
	PolicyRejected
)

const (
	maxNameBytes     = 64
	maxTools         = 64
	maxParameters    = 32
	maxArgumentBytes = 16384
	maxDepth         = 8
	maxObjectMembers = 64
	maxArrayElements = 128
	maxValues        = 1024
	maxKeyBytes      = 128
	maxStringBytes   = 4096
	maxNumberBytes   = 64
)

// Catalog owns an immutable snapshot of caller-authorized definitions. It is
// safe for concurrent admissions. A zero-value Catalog is an empty deny-all
// catalog; a model proposal must never be used to populate authority implicitly.
type Catalog struct {
	definitions map[string]definition
}

type definition struct {
	parameters map[string]Parameter
}

// Admission owns only a decision and, when admitted, an exact call snapshot.
// Its zero value is Rejected / NotEvaluated. It grants no execution capability.
type Admission struct {
	status Status
	reason Reason
	call   Call
}
