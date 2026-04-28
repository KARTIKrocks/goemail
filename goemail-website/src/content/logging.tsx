import CodeBlock from '../components/CodeBlock';

export default function LoggingDocs() {
  return (
    <section id="logging" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Logging</h2>
      <p className="text-text-muted mb-4">
        goemail does not pull in a logging library. Instead it defines a small <code>Logger</code>
        interface that mirrors structured-logging conventions and ships an adapter for the standard
        library's <code>slog</code>. Plug in your existing logger by implementing four methods.
      </p>

      <h3 id="logging-interface" className="text-lg font-semibold text-text-heading mt-8 mb-2">Logger Interface</h3>
      <CodeBlock code={`type Logger interface {
    Debug(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
    Warn(msg string, keysAndValues ...any)
    Error(msg string, keysAndValues ...any)
    With(keysAndValues ...any) Logger
}`} />
      <p className="text-text-muted mb-3 text-sm">
        <code>With</code> returns a child logger with extra structured fields attached — handy for
        threading request IDs through middleware.
      </p>

      <h3 id="logging-slog" className="text-lg font-semibold text-text-heading mt-8 mb-2">slog</h3>
      <p className="text-text-muted mb-3">
        For Go 1.21+, use <code>NewSlogLogger</code> to wrap any <code>*slog.Logger</code>:
      </p>
      <CodeBlock code={`import (
    "log/slog"
    "os"

    email "github.com/KARTIKrocks/goemail"
)

slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

config := email.SMTPConfig{
    Host:   "smtp.gmail.com",
    Logger: email.NewSlogLogger(slogLogger),
}`} />

      <h3 id="logging-custom" className="text-lg font-semibold text-text-heading mt-8 mb-2">Custom Loggers</h3>
      <p className="text-text-muted mb-3">
        Wrapping zap, zerolog, or logrus is a small adapter type. Here is the minimal shape — fill in
        the bodies with your logger of choice:
      </p>
      <CodeBlock code={`type myLogger struct{ /* underlying logger */ }

func (l myLogger) Debug(msg string, kv ...any)    { /* ... */ }
func (l myLogger) Info(msg string, kv ...any)     { /* ... */ }
func (l myLogger) Warn(msg string, kv ...any)     { /* ... */ }
func (l myLogger) Error(msg string, kv ...any)    { /* ... */ }
func (l myLogger) With(kv ...any) email.Logger    { return l /* with fields */ }`} />
    </section>
  );
}
