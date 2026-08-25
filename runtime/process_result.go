package runtime

// ProcessResult is the completed direct-child result. A non-zero ExitCode is
// an application result, not an infrastructure failure.
type ProcessResult struct {
	ExitCode   int
	Canceled   bool
	Terminated bool

	Stdout []byte
	Stderr []byte

	StdoutTruncated bool
	StderrTruncated bool
}

func (r ProcessResult) clone() ProcessResult {
	r.Stdout = append([]byte(nil), r.Stdout...)
	r.Stderr = append([]byte(nil), r.Stderr...)
	return r
}
