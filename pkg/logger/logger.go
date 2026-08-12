package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Logger wraps the standard library logger.
type Logger struct {
	level  Level
	logger *log.Logger
}

// New creates a new logger.
func New(cfg Config) *Logger {
	return &Logger{
		level: cfg.Level,
		logger: log.New(
			os.Stdout,
			"",
			log.LstdFlags,
		),
	}
}

func (l *Logger) log(level Level, msg string, fields []Field) {
	if level < l.level {
		return
	}

	if len(fields) == 0 {
		l.logger.Printf("[%s] %s", level.String(), msg)
		return
	}

	l.logger.Printf("[%s] %s %s", level.String(), msg, formatFields(fields))
}

func formatFields(fields []Field) string {
	if len(fields) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, field := range fields {
		if i > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(field.Key)
		builder.WriteByte('=')
		builder.WriteString(formatFieldValue(field.Value))
	}

	return builder.String()
}

func formatFieldValue(value any) string {
	if value == nil {
		return "<nil>"
	}

	return fmt.Sprint(value)
}

func (l *Logger) Debug(msg string) {
	l.log(Debug, msg, nil)
}

func (l *Logger) Info(msg string) {
	l.log(Info, msg, nil)
}

func (l *Logger) Warn(msg string) {
	l.log(Warn, msg, nil)
}

func (l *Logger) Error(msg string) {
	l.log(Error, msg, nil)
}

func (l *Logger) DebugFields(msg string, fields ...Field) {
	l.log(Debug, msg, fields)
}

func (l *Logger) InfoFields(msg string, fields ...Field) {
	l.log(Info, msg, fields)
}

func (l *Logger) WarnFields(msg string, fields ...Field) {
	l.log(Warn, msg, fields)
}

func (l *Logger) ErrorFields(msg string, fields ...Field) {
	l.log(Error, msg, fields)
}
