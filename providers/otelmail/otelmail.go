package otelmail

import (
	"context"
	"strings"

	email "github.com/KARTIKrocks/goemail"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultTracerName = "github.com/KARTIKrocks/goemail"
	defaultSpanName   = "email.send"
)

type config struct {
	tracerProvider trace.TracerProvider
	tracerName     string
	spanName       string
}

// Option configures the tracing middleware.
type Option func(*config)

// WithTracerProvider sets a custom TracerProvider.
// By default the global provider from otel.GetTracerProvider() is used.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *config) {
		c.tracerProvider = tp
	}
}

// WithTracerName sets the name used when creating the tracer.
// Default: "github.com/KARTIKrocks/goemail".
func WithTracerName(name string) Option {
	return func(c *config) {
		c.tracerName = name
	}
}

// WithSpanName sets the name of the span created for each send.
// Default: "email.send".
func WithSpanName(name string) Option {
	return func(c *config) {
		c.spanName = name
	}
}

// WithTracing returns an email.Middleware that creates OpenTelemetry spans
// around each Send call.
func WithTracing(opts ...Option) email.Middleware {
	cfg := config{
		tracerName: defaultTracerName,
		spanName:   defaultSpanName,
	}
	for _, o := range opts {
		o(&cfg)
	}

	tp := cfg.tracerProvider
	if tp == nil {
		tp = otel.GetTracerProvider()
	}
	tracer := tp.Tracer(cfg.tracerName)

	return func(next email.Sender) email.Sender {
		return &tracingSender{
			next:     next,
			tracer:   tracer,
			spanName: cfg.spanName,
		}
	}
}

type tracingSender struct {
	next     email.Sender
	tracer   trace.Tracer
	spanName string
}

func (s *tracingSender) Send(ctx context.Context, e *email.Email) error {
	ctx, span := s.tracer.Start(ctx, s.spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("email.from", e.From),
			attribute.String("email.to", strings.Join(e.To, ",")),
			attribute.String("email.subject", e.Subject),
			attribute.Int("email.recipients.count", len(e.To)+len(e.Cc)+len(e.Bcc)),
		),
	)
	defer span.End()

	err := s.next.Send(ctx, e)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (s *tracingSender) Close() error {
	return s.next.Close()
}
