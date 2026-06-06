package ses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	email "github.com/KARTIKrocks/goemail"
)

// sesAPI abstracts the SES v2 client for testability.
type sesAPI interface {
	SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

// Config holds the SES provider configuration.
type Config struct {
	// Region is the AWS region (e.g., "us-east-1").
	// Falls back to the SDK default if empty.
	Region string

	// ConfigurationSetName is an optional SES configuration set name.
	ConfigurationSetName string

	// AWSConfig overrides the full AWS SDK configuration.
	// If nil, aws.Config is loaded from the default credential chain.
	AWSConfig *aws.Config
}

// Sender sends emails through AWS SES v2 using raw MIME messages.
type Sender struct {
	client sesAPI
	cfgSet string
}

// New creates a new SES Sender. It loads AWS credentials from the default
// chain unless Config.AWSConfig is provided.
func New(ctx context.Context, cfg Config) (*Sender, error) {
	var awsCfg aws.Config
	var err error

	if cfg.AWSConfig != nil {
		awsCfg = *cfg.AWSConfig
	} else {
		opts := []func(*awsconfig.LoadOptions) error{}
		if cfg.Region != "" {
			opts = append(opts, awsconfig.WithRegion(cfg.Region))
		}
		awsCfg, err = awsconfig.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("ses: load AWS config: %w", err)
		}
	}

	client := sesv2.NewFromConfig(awsCfg)
	return newSenderWithClient(client, cfg.ConfigurationSetName), nil
}

// newSenderWithClient creates a Sender with a custom sesAPI implementation.
// Used internally for testing.
func newSenderWithClient(client sesAPI, cfgSet string) *Sender {
	return &Sender{client: client, cfgSet: cfgSet}
}

// Send sends an email through SES using a raw MIME message.
// It implements email.Sender.
func (s *Sender) Send(ctx context.Context, e *email.Email) error {
	raw, err := email.BuildRawMessage(e)
	if err != nil {
		return fmt.Errorf("ses: build raw message: %w", err)
	}

	// Gather all destinations
	dest := &types.Destination{}
	dest.ToAddresses = e.To
	if len(e.Cc) > 0 {
		dest.CcAddresses = e.Cc
	}
	if len(e.Bcc) > 0 {
		dest.BccAddresses = e.Bcc
	}

	input := &sesv2.SendEmailInput{
		Content: &types.EmailContent{
			Raw: &types.RawMessage{
				Data: raw,
			},
		},
		Destination:      dest,
		FromEmailAddress: aws.String(e.From),
	}

	if s.cfgSet != "" {
		input.ConfigurationSetName = aws.String(s.cfgSet)
	}

	_, err = s.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("ses: send email: %w", err)
	}

	return nil
}

// Close is a no-op for SES (HTTP-based, no persistent connections).
func (s *Sender) Close() error { return nil }
