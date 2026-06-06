// Package mailgun provides a Mailgun provider adapter for the goemail library.
//
// It implements the email.Sender interface using the Mailgun v3 Messages API,
// allowing you to send emails through Mailgun without pulling in any
// external SDK dependencies.
//
// # Usage
//
//	sender, err := mailgun.New(mailgun.Config{
//	    Domain: "mg.example.com",
//	    APIKey: os.Getenv("MAILGUN_API_KEY"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer sender.Close()
//
//	e, _ := email.NewEmail().
//	    SetFrom("sender@mg.example.com").
//	    AddTo("recipient@example.com").
//	    SetSubject("Hello from Mailgun").
//	    SetBody("Plain text body").
//	    Build()
//
//	err = sender.Send(context.Background(), e)
//
// # EU Region
//
// For EU-based Mailgun accounts, set the BaseURL:
//
//	sender, _ := mailgun.New(mailgun.Config{
//	    Domain:  "mg.example.com",
//	    APIKey:  os.Getenv("MAILGUN_API_KEY"),
//	    BaseURL: "https://api.eu.mailgun.net",
//	})
package mailgun
