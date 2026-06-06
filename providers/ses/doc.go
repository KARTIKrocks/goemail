// Package ses provides an AWS SES v2 provider adapter for the goemail library.
//
// It implements the email.Sender interface using the AWS SDK for Go v2
// SESv2 SendEmail API with raw messages, allowing you to send emails
// through Amazon Simple Email Service.
//
// # Usage
//
//	sender, err := ses.New(context.Background(), ses.Config{
//	    Region: "us-east-1",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer sender.Close()
//
//	e, _ := email.NewEmail().
//	    SetFrom("sender@example.com").
//	    AddTo("recipient@example.com").
//	    SetSubject("Hello from SES").
//	    SetBody("Plain text body").
//	    Build()
//
//	err = sender.Send(context.Background(), e)
//
// # AWS Credentials
//
// The adapter uses the standard AWS SDK credential chain
// (environment variables, shared config, IAM role, etc.).
// You can also pass a pre-configured aws.Config via Config.AWSConfig.
package ses
