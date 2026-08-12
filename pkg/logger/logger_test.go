package logger

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
)

var _ Contract = (*Logger)(nil)
var _ Contract = (*testLogger)(nil)

type testLogger struct {
	debug string
	info  string
	warn  string
	err   string
}

type oldLogger struct{}

func (t *testLogger) Debug(msg string) { t.debug = msg }
func (t *testLogger) Info(msg string)  { t.info = msg }
func (t *testLogger) Warn(msg string)  { t.warn = msg }
func (t *testLogger) Error(msg string) { t.err = msg }

func (oldLogger) Debug(msg string) {}
func (oldLogger) Info(msg string)  {}
func (oldLogger) Warn(msg string)  {}
func (oldLogger) Error(msg string) {}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{Debug, "DEBUG"},
		{Info, "INFO"},
		{Warn, "WARN"},
		{Error, "ERROR"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("expected %s, got %s", tt.want, got)
		}
	}
}

func TestLoggerWritesExpectedOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := &Logger{
		level:  Debug,
		logger: log.New(buf, "", 0),
	}

	logger.Info("hello")
	got := buf.String()
	if !strings.Contains(got, "[INFO] hello") {
		t.Fatalf("expected log output to contain %q, got %q", "[INFO] hello", got)
	}
}

func TestLoggerRespectsLevelFiltering(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := &Logger{
		level:  Warn,
		logger: log.New(buf, "", 0),
	}

	logger.Info("skip")
	if buf.Len() != 0 {
		t.Fatalf("expected no output for Info at Warn level, got %q", buf.String())
	}

	logger.Error("keep")
	if !strings.Contains(buf.String(), "[ERROR] keep") {
		t.Fatalf("expected log output to contain %q, got %q", "[ERROR] keep", buf.String())
	}
}

func TestFieldConstruction(t *testing.T) {
	field := Field{Key: "user", Value: "alice"}
	if field.Key != "user" || field.Value != "alice" {
		t.Fatalf("unexpected field data: %#v", field)
	}
}

func TestLoggerStructuredFieldsOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := &Logger{
		level:  Debug,
		logger: log.New(buf, "", 0),
	}

	logger.InfoFields("started", Field{Key: "user", Value: "alice"}, Field{Key: "id", Value: 42})
	want := "[INFO] started user=alice id=42"
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestLoggerStructuredFieldsWithNilAndEmptyValues(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := &Logger{
		level:  Debug,
		logger: log.New(buf, "", 0),
	}

	logger.ErrorFields("failed", Field{Key: "reason", Value: "timeout"}, Field{Key: "retry", Value: nil}, Field{Key: "empty", Value: ""})
	want := "[ERROR] failed reason=timeout retry=<nil> empty="
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestLoggerStructuredFieldOrderingAndDuplicateKeys(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := &Logger{
		level:  Debug,
		logger: log.New(buf, "", 0),
	}

	logger.WarnFields("check", Field{Key: "id", Value: 1}, Field{Key: "id", Value: 2}, Field{Key: "user", Value: "bob"})
	want := "[WARN] check id=1 id=2 user=bob"
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestStructuredContractCompatibility(t *testing.T) {
	var _ Contract = (*Logger)(nil)
	var _ StructuredContract = (*Logger)(nil)

	var loggerContract Contract = oldLogger{}
	loggerContract.Info("legacy")
	if _, ok := loggerContract.(StructuredContract); ok {
		t.Fatalf("legacy Contract should not implement StructuredContract")
	}
}

func TestConcurrentStructuredLogging(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := &Logger{
		level:  Debug,
		logger: log.New(buf, "", 0),
	}

	const workers = 50
	const iterations = 20
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				logger.InfoFields("event", Field{Key: "worker", Value: worker}, Field{Key: "index", Value: j})
			}
		}(i)
	}

	wg.Wait()

	if buf.Len() == 0 {
		t.Fatal("expected concurrent logs to write output")
	}
}
