// Package logger provides logging utilities for Forge.
//
// Contract is the minimal logging abstraction for Forge components.
// Implementations may provide the logging backend, while the concrete
// *Logger remains the default implementation in this package.
//
// Structured fields will be introduced in FW-025.2.
package logger

// Contract defines the minimal logging API Forge components may depend on.
//
// It intentionally omits structured logging, context awareness, and other
// advanced features until FW-025.2 and later work.
type Contract interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}
