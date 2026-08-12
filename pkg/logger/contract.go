// Package logger provides logging utilities for Forge.
//
// Contract is the minimal logging abstraction for Forge components.
// Implementations may provide the logging backend, while the concrete
// *Logger remains the default implementation in this package.
//
// FW-025.2 adds lightweight structured logging support through Field values
// and an optional structured logging extension interface.
package logger

// Field represents a single structured logging field.
//
// Values are rendered using Go sprint semantics and nil is rendered as
// <nil>. Field ordering is preserved exactly as supplied. Duplicate keys are
// allowed and emitted in order.
type Field struct {
	Key   string
	Value any
}

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

// StructuredContract is an optional extension for implementations that support
// structured logging fields. It preserves backward compatibility with older
// Contract implementations.
type StructuredContract interface {
	Contract
	DebugFields(msg string, fields ...Field)
	InfoFields(msg string, fields ...Field)
	WarnFields(msg string, fields ...Field)
	ErrorFields(msg string, fields ...Field)
}
