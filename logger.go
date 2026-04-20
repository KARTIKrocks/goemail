package email

// Logger is the interface for logging within the email package.
// Users can provide their own implementation or use the provided adapters
// for popular logging libraries (slog, zap, logrus).
//
// By default, if no logger is provided, logging is disabled (NoOpLogger).
//
// keysAndValues must be supplied as alternating key/value pairs. An odd-length
// slice is backend-dependent: slog, for example, renders the dangling key as
// "!BADKEY". Always pass pairs.
type Logger interface {
	// Debug logs a debug message with optional key-value pairs
	Debug(msg string, keysAndValues ...any)

	// Info logs an info message with optional key-value pairs
	Info(msg string, keysAndValues ...any)

	// Warn logs a warning message with optional key-value pairs
	Warn(msg string, keysAndValues ...any)

	// Error logs an error message with optional key-value pairs
	Error(msg string, keysAndValues ...any)

	// With returns a new logger with the given key-value pairs added to context
	With(keysAndValues ...any) Logger
}

// NoOpLogger is a logger that discards all logs.
// This is the default logger when none is provided.
type NoOpLogger struct{}

// Debug implements Logger
func (NoOpLogger) Debug(_ string, _ ...any) {}

// Info implements Logger
func (NoOpLogger) Info(_ string, _ ...any) {}

// Warn implements Logger
func (NoOpLogger) Warn(_ string, _ ...any) {}

// Error implements Logger
func (NoOpLogger) Error(_ string, _ ...any) {}

// With implements Logger
func (n NoOpLogger) With(_ ...any) Logger { return n }
