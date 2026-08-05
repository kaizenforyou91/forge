package logger

import "testing"

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