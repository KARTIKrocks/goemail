package email

import "log/slog"

// SlogLogger wraps slog.Logger to implement the email.Logger interface.
// This adapter allows using Go's standard library structured logger (slog)
// with the email package.
//
// Example:
//
//	import "log/slog"
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	config := email.SMTPConfig{
//	    Host:   "smtp.gmail.com",
//	    Logger: email.NewSlogLogger(logger),
//	}
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates a new Logger from slog.Logger
func NewSlogLogger(logger *slog.Logger) Logger {
	return &SlogLogger{logger: logger}
}

// Debug implements Logger
func (l *SlogLogger) Debug(msg string, keysAndValues ...any) {
	l.logger.Debug(msg, keysAndValues...)
}

// Info implements Logger
func (l *SlogLogger) Info(msg string, keysAndValues ...any) {
	l.logger.Info(msg, keysAndValues...)
}

// Warn implements Logger
func (l *SlogLogger) Warn(msg string, keysAndValues ...any) {
	l.logger.Warn(msg, keysAndValues...)
}

// Error implements Logger
func (l *SlogLogger) Error(msg string, keysAndValues ...any) {
	l.logger.Error(msg, keysAndValues...)
}

// With implements Logger
func (l *SlogLogger) With(keysAndValues ...any) Logger {
	return &SlogLogger{logger: l.logger.With(keysAndValues...)}
}
