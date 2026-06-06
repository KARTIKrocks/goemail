// Package otelmail provides OpenTelemetry tracing middleware for the goemail library.
//
// It returns an [email.Middleware] that creates a span for every Send call,
// recording email attributes and error status. Because it lives in a separate
// Go module, the core goemail library stays dependency-free — users who do
// not use OpenTelemetry never pull in the OTel SDK.
//
// # Usage
//
//	import "github.com/KARTIKrocks/goemail/providers/otelmail"
//
//	wrapped := email.Chain(sender,
//	    otelmail.WithTracing(),
//	    email.WithLogging(logger),
//	)
//
// # Span Attributes
//
// Each span includes:
//   - email.from — sender address
//   - email.to — comma-separated To recipients
//   - email.subject — email subject line
//   - email.recipients.count — total recipient count (To + Cc + Bcc)
//
// # Options
//
// Use [WithTracerProvider], [WithTracerName], or [WithSpanName] to customise
// the tracer and span names.
package otelmail
