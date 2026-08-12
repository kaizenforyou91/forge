// Package logger provides logging utilities for Forge.
//
// Contract is the minimal logging abstraction for Forge components.
// Implementations may provide the logging backend, while the concrete
// *Logger remains the default implementation in this package.
//
// FW-025.2 adds lightweight structured logging support through Field values
// and an optional StructuredContract extension.
package logger
