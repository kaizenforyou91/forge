package logger

import (
	"bytes"
	"log"
	"strings"
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

func (t *testLogger) Debug(msg string) { t.debug = msg }
func (t *testLogger) Info(msg string)  { t.info = msg }
func (t *testLogger) Warn(msg string)  { t.warn = msg }
func (t *testLogger) Error(msg string) { t.err = msg }

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
