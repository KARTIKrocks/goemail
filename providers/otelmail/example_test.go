package otelmail_test

import (
	"context"
	"fmt"

	email "github.com/KARTIKrocks/goemail"
	"github.com/KARTIKrocks/goemail/providers/otelmail"
)

func Example() {
	// Use any email.Sender (SMTP, mock, etc.)
	var sender email.Sender = email.NewMockSender()

	// Wrap with OTel tracing (and any other middleware)
	wrapped := email.Chain(sender,
		otelmail.WithTracing(), // creates a span per Send
	)
	defer wrapped.Close()

	e := email.NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetSubject("Hello from otelmail").
		SetBody("Tracing is enabled!")

	if err := wrapped.Send(context.Background(), e); err != nil {
		fmt.Println("send failed:", err)
		return
	}
	fmt.Println("sent")
	// Output: sent
}
