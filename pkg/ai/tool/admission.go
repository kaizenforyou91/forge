package tool

import (
	"bytes"
	"strings"
)

// Admit evaluates one proposal synchronously and performs zero execution.
// Decision order is receiver, name, lookup, byte limit, lexical JSON checks,
// bounded traversal, then root schema. Ordinary rejection returns nil error.
// All argument bytes (including whitespace) count toward the 16384-byte cap.
// Containers have depth <=8 (root=1), objects <=64 members, arrays <=128
// elements, and the document <=1024 values (including containers, excluding
// keys). Decoded keys are <=128 UTF-8 bytes, string values <=4096 bytes, and
// number tokens <=64 bytes. Root parameters are closed and exactly typed.
// An admitted snapshot retains original bytes, never repaired/reserialized JSON.
func (c *Catalog) Admit(call Call) (Admission, error) {
	if c == nil {
		return Admission{}, ErrInvalidCatalog
	}
	reject := func(reason Reason) (Admission, error) {
		return Admission{reason: reason}, nil
	}
	if !validName(call.Name) {
		return reject(MalformedCall)
	}
	selected, exists := c.definitions[call.Name]
	if !exists {
		return reject(UnknownTool)
	}
	if len(call.Arguments) > maxArgumentBytes {
		return reject(PolicyRejected)
	}
	snapshot := bytes.Clone(call.Arguments)
	if !validJSONLexically(snapshot) {
		return reject(InvalidArguments)
	}
	fields, reason := inspectArguments(snapshot)
	if reason != Allowed {
		return reject(reason)
	}
	for name, observed := range fields {
		parameter, exists := selected.parameters[name]
		if !exists || parameter.Type != observed {
			return reject(InvalidArguments)
		}
	}
	for name, parameter := range selected.parameters {
		if _, present := fields[name]; parameter.Required && !present {
			return reject(InvalidArguments)
		}
	}
	return Admission{
		status: Admitted,
		reason: Allowed,
		call:   Call{Name: strings.Clone(call.Name), Arguments: snapshot},
	}, nil
}

// Status reports admission, never execution.
func (a Admission) Status() Status { return a.status }

// Reason reports the deterministic classification.
func (a Admission) Reason() Reason { return a.reason }

// AdmittedCall returns an independent argument copy on every successful access.
// Rejected and zero-value admissions expose no call or payload. The returned
// data is not an execution permit; a future execution layer needs its own checks.
func (a Admission) AdmittedCall() (Call, bool) {
	if a.status != Admitted || a.reason != Allowed {
		return Call{}, false
	}
	return Call{Name: a.call.Name, Arguments: bytes.Clone(a.call.Arguments)}, true
}
