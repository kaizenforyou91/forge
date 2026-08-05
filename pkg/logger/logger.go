package logger

import (
	"log"
	"os"
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

func (l *Logger) log(level Level, msg string) {
	if level < l.level {
		return
	}

	l.logger.Printf("[%s] %s", level.String(), msg)
}

func (l *Logger) Debug(msg string) {
	l.log(Debug, msg)
}

func (l *Logger) Info(msg string) {
	l.log(Info, msg)
}

func (l *Logger) Warn(msg string) {
	l.log(Warn, msg)
}

func (l *Logger) Error(msg string) {
	l.log(Error, msg)
}
