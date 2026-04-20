package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	email "github.com/KARTIKrocks/goemail"
)

// simpleMetrics is a basic MetricsCollector for demonstration.
type simpleMetrics struct {
	attempts  atomic.Int64
	successes atomic.Int64
	failures  atomic.Int64
}

func (m *simpleMetrics) IncSendAttempt() { m.attempts.Add(1) }
func (m *simpleMetrics) IncSendSuccess() { m.successes.Add(1) }
func (m *simpleMetrics) IncSendFailure() { m.failures.Add(1) }
func (m *simpleMetrics) ObserveSendDuration(d time.Duration) {
	fmt.Printf("  [metrics] send took %s\n", d)
}

func main() {
	// Create a mock sender for demonstration
	mock := email.NewMockSender()

	// Set up structured logger
	logger := email.NewSlogLogger(
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
	)

	// Set up metrics collector
	metrics := &simpleMetrics{}

	// Set up lifecycle hooks
	hooks := email.SendHooks{
		OnSend: func(_ context.Context, e *email.Email) {
			fmt.Printf("  [hook] about to send to %v\n", e.To)
		},
		OnSuccess: func(_ context.Context, e *email.Email, d time.Duration) {
			fmt.Printf("  [hook] sent successfully in %s\n", d)
		},
		OnFailure: func(_ context.Context, e *email.Email, d time.Duration, err error) {
			fmt.Printf("  [hook] send failed in %s: %v\n", d, err)
		},
	}

	// Option A: Use Chain to wrap the sender directly
	wrapped := email.Chain(mock,
		email.WithRecovery(),       // Outermost: catch panics
		email.WithLogging(logger),  // Log all sends
		email.WithHooks(hooks),     // Lifecycle callbacks
		email.WithMetrics(metrics), // Record metrics
	)
	mailer := email.NewMailer(wrapped, "sender@example.com")

	// Option B: Equivalent using NewMailerWithOptions
	// mailer := email.NewMailerWithOptions(mock, "sender@example.com",
	// 	email.WithMiddleware(
	// 		email.WithRecovery(),
	// 		email.WithLogging(logger),
	// 		email.WithHooks(hooks),
	// 		email.WithMetrics(metrics),
	// 	),
	// )

	ctx := context.Background()

	fmt.Println("--- Sending email 1 ---")
	if err := mailer.Send(ctx, []string{"alice@example.com"}, "Hello Alice", "Welcome!"); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\n--- Sending email 2 ---")
	if err := mailer.Send(ctx, []string{"bob@example.com"}, "Hello Bob", "Welcome!"); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Printf("\n--- Metrics Summary ---\n")
	fmt.Printf("Attempts:  %d\n", metrics.attempts.Load())
	fmt.Printf("Successes: %d\n", metrics.successes.Load())
	fmt.Printf("Failures:  %d\n", metrics.failures.Load())

	mailer.Close()
}
