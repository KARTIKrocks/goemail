// Package sendgrid provides a SendGrid provider adapter for the goemail library.
//
// It implements the email.Sender interface using the SendGrid v3 Web API,
// allowing you to send emails through SendGrid without pulling in any
// external SDK dependencies.
//
// # Usage
//
//	sender, err := sendgrid.New(sendgrid.Config{
//	    APIKey: os.Getenv("SENDGRID_API_KEY"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer sender.Close()
//
//	e, _ := email.NewEmail().
//	    SetFrom("sender@example.com").
//	    AddTo("recipient@example.com").
//	    SetSubject("Hello from SendGrid").
//	    SetBody("Plain text body").
//	    Build()
//
//	err = sender.Send(context.Background(), e)
package sendgrid
