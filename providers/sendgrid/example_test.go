package sendgrid_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	email "github.com/KARTIKrocks/goemail"
	"github.com/KARTIKrocks/goemail/providers/sendgrid"
)

func Example() {
	// Create a test server to stand in for the SendGrid API.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	sender, err := sendgrid.New(sendgrid.Config{
		APIKey:  "test-api-key",
		BaseURL: srv.URL,
	})
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	defer sender.Close()

	e, _ := email.NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetSubject("Hello from SendGrid").
		SetBody("This is a test email.").
		Build()

	if err := sender.Send(context.Background(), e); err != nil {
		fmt.Printf("send error: %v\n", err)
		return
	}

	fmt.Println("email sent successfully")
	// Output: email sent successfully
}
