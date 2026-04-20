package email

import (
	"context"
	"net/http"
	"time"
)

// EventType represents the type of a webhook delivery notification event.
type EventType string

const (
	// EventDelivered indicates the message was accepted by the recipient's mail server.
	EventDelivered EventType = "delivered"

	// EventBounced indicates a hard bounce (permanent delivery failure).
	EventBounced EventType = "bounced"

	// EventDeferred indicates a soft bounce (temporary delivery failure).
	EventDeferred EventType = "deferred"

	// EventOpened indicates the recipient opened the email.
	EventOpened EventType = "opened"

	// EventClicked indicates the recipient clicked a link in the email.
	EventClicked EventType = "clicked"

	// EventComplained indicates the recipient marked the email as spam.
	EventComplained EventType = "complained"

	// EventUnsubscribed indicates the recipient unsubscribed.
	EventUnsubscribed EventType = "unsubscribed"

	// EventDropped indicates the provider rejected the message before sending.
	EventDropped EventType = "dropped"
)

// WebhookEvent is the provider-agnostic representation of a delivery
// notification event. Provider-specific parsers (in providers/ submodules)
// normalize raw webhook payloads into this type.
type WebhookEvent struct {
	// Type is the normalized event type.
	Type EventType

	// MessageID is the provider-assigned message identifier.
	MessageID string

	// Recipient is the email address this event relates to.
	Recipient string

	// Timestamp is when the event occurred at the provider.
	Timestamp time.Time

	// Provider identifies the source (e.g. "sendgrid", "mailgun", "ses").
	Provider string

	// Reason contains detail for bounces, drops, or deferrals.
	// Empty for positive events like delivered/opened.
	Reason string

	// URL is the clicked URL for EventClicked events. Empty otherwise.
	URL string

	// UserAgent is the user agent for open/click events when available.
	UserAgent string

	// IP is the IP address associated with the event when available.
	IP string

	// Tags contains provider-specific tags/categories for the message.
	Tags []string

	// Metadata holds arbitrary provider-specific key-value data that doesn't
	// map to a named field.
	Metadata map[string]string

	// RawPayload is the original bytes from the provider, preserved for
	// debugging or provider-specific processing.
	RawPayload []byte
}

// WebhookParser parses provider-specific HTTP requests into WebhookEvents.
// Each provider adapter (providers/webhooksendgrid, etc.) implements this
// interface. Implementations must be safe for concurrent use.
type WebhookParser interface {
	// Parse reads the HTTP request and returns zero or more normalized events.
	// Providers like SendGrid batch multiple events per request.
	Parse(r *http.Request) ([]WebhookEvent, error)
}

// WebhookHandler handles normalized webhook events.
// Implementations must be safe for concurrent use.
type WebhookHandler interface {
	// HandleEvent processes a single webhook event.
	// Returning an error causes the WebhookReceiver to respond with 500,
	// signaling the provider to retry delivery.
	HandleEvent(ctx context.Context, event WebhookEvent) error
}

// WebhookHandlerFunc is an adapter to allow ordinary functions to be used
// as WebhookHandlers, following the net/http.HandlerFunc pattern.
type WebhookHandlerFunc func(ctx context.Context, event WebhookEvent) error

// HandleEvent calls f(ctx, event).
func (f WebhookHandlerFunc) HandleEvent(ctx context.Context, event WebhookEvent) error {
	return f(ctx, event)
}

// WebhookReceiver is an [http.Handler] that receives webhook POSTs from
// email providers, parses them using a [WebhookParser], and dispatches
// normalized events to a [WebhookHandler].
//
// It is safe for concurrent use.
//
//	receiver := email.NewWebhookReceiver(parser, handler)
//	http.Handle("/webhooks/email", receiver)
type WebhookReceiver struct {
	parser  WebhookParser
	handler WebhookHandler
	logger  Logger
	filter  map[EventType]struct{}
}

// WebhookReceiverOption configures a [WebhookReceiver].
type WebhookReceiverOption func(*WebhookReceiver)

// WithWebhookLogger sets the logger for the webhook receiver.
func WithWebhookLogger(l Logger) WebhookReceiverOption {
	return func(wr *WebhookReceiver) {
		if l != nil {
			wr.logger = l
		}
	}
}

// WithEventFilter restricts the receiver to only dispatch the listed event
// types. Events not in the filter are silently acknowledged (200 OK) but
// not dispatched to the handler.
func WithEventFilter(types ...EventType) WebhookReceiverOption {
	return func(wr *WebhookReceiver) {
		wr.filter = make(map[EventType]struct{}, len(types))
		for _, t := range types {
			wr.filter[t] = struct{}{}
		}
	}
}

// NewWebhookReceiver creates a new [WebhookReceiver].
func NewWebhookReceiver(parser WebhookParser, handler WebhookHandler, opts ...WebhookReceiverOption) *WebhookReceiver {
	wr := &WebhookReceiver{
		parser:  parser,
		handler: handler,
		logger:  NoOpLogger{},
	}
	for _, opt := range opts {
		opt(wr)
	}
	return wr
}

// ServeHTTP implements [http.Handler]. It parses the request, dispatches
// each event to the handler, and responds:
//   - 200 OK if all events are handled successfully
//   - 400 Bad Request if the request cannot be parsed
//   - 405 Method Not Allowed if the method is not POST
//   - 500 Internal Server Error if any handler returns an error
//
// All events in the batch are dispatched even if one fails — the receiver
// does not short-circuit on the first error. This avoids dropping later
// events in a batch when a transient error occurs in the middle, but it
// means handlers MUST be idempotent: a 500 response causes the provider
// to redeliver the entire batch, including events that already succeeded.
func (wr *WebhookReceiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	events, err := wr.parser.Parse(r)
	if err != nil {
		wr.logger.Error("webhook parse failed", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	wr.logger.Debug("webhook received", "event_count", len(events))

	failures := 0
	for i := range events {
		if err := wr.dispatchEvent(r.Context(), events[i]); err != nil {
			failures++
		}
	}

	if failures > 0 {
		wr.logger.Error("webhook batch had handler failures",
			"failures", failures,
			"total", len(events),
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// dispatchEvent checks the filter and dispatches a single event.
func (wr *WebhookReceiver) dispatchEvent(ctx context.Context, event WebhookEvent) error {
	if wr.filter != nil {
		if _, ok := wr.filter[event.Type]; !ok {
			return nil
		}
	}

	if err := wr.handler.HandleEvent(ctx, event); err != nil {
		wr.logger.Error("webhook handler failed",
			"event_type", string(event.Type),
			"message_id", event.MessageID,
			"error", err,
		)
		return err
	}
	return nil
}
