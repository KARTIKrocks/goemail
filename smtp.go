package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"net"
	"net/smtp"
	"net/textproto"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultTimeout is the default timeout for SMTP operations
	DefaultTimeout = 30 * time.Second

	// DefaultMaxRetries is the default number of retry attempts
	DefaultMaxRetries = 3

	// DefaultRetryDelay is the default initial retry delay
	DefaultRetryDelay = time.Second

	// DefaultRetryBackoff is the default exponential backoff multiplier
	DefaultRetryBackoff = 2.0

	// DefaultRateLimit is the default rate limit (emails per second)
	DefaultRateLimit = 10
)

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	// Host is the SMTP server hostname
	Host string

	// Port is the SMTP server port (typically 587 for TLS, 465 for SSL)
	Port int

	// Username is the SMTP authentication username
	Username string

	// Password is the SMTP authentication password
	Password string

	// From is the default sender email address
	From string

	// UseTLS enables STARTTLS encryption
	UseTLS bool

	// Timeout is the connection timeout (default: 30s)
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts (default: 3).
	// Set to a negative value to disable retries.
	MaxRetries int

	// RetryDelay is the initial retry delay (default: 1s)
	RetryDelay time.Duration

	// RetryBackoff is the exponential backoff multiplier (default: 2.0)
	RetryBackoff float64

	// RateLimit is the maximum number of emails per second (default: 10).
	// Set to a negative value to disable rate limiting.
	RateLimit int

	// PoolSize is the maximum number of open SMTP connections in the pool.
	// 0 (default) disables pooling — each Send dials a fresh connection.
	PoolSize int

	// MaxIdleConns is the maximum number of idle connections kept in the pool.
	// Default: 2. Only used when PoolSize > 0.
	MaxIdleConns int

	// PoolMaxLifetime is the maximum lifetime of a pooled connection.
	// Connections older than this are discarded on checkout. Default: 30m.
	PoolMaxLifetime time.Duration

	// PoolMaxIdleTime is the maximum idle time before a connection is evicted.
	// Default: 5m.
	PoolMaxIdleTime time.Duration

	// MaxMessages is the maximum number of messages sent on a single connection
	// before it is rotated. Default: 100.
	MaxMessages int

	// PoolWaitTimeout is the maximum time to wait for a connection when the
	// pool is exhausted. Default: 5s.
	PoolWaitTimeout time.Duration

	// Logger is the logger interface for observability
	// If nil, logging is disabled (NoOpLogger used)
	Logger Logger

	// DKIM is the optional DKIM signing configuration.
	// If non-nil, all outgoing messages are signed with a DKIM-Signature header.
	DKIM *DKIMConfig
}

// Validate validates the SMTP configuration
func (c SMTPConfig) Validate() error {
	if c.Host == "" {
		return errors.New("smtp: host is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("smtp: invalid port %d (must be 1-65535)", c.Port)
	}
	if err := c.validateCredentials(); err != nil {
		return err
	}
	if err := c.validateNonNegative(); err != nil {
		return err
	}
	if c.PoolSize > 0 && c.MaxIdleConns > c.PoolSize {
		return errors.New("smtp: max idle conns cannot exceed pool size")
	}
	if c.DKIM != nil {
		if err := c.DKIM.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c SMTPConfig) validateCredentials() error {
	if c.Username == "" && c.Password != "" {
		return errors.New("smtp: password set but username is empty")
	}
	if c.Password == "" && c.Username != "" {
		return errors.New("smtp: username set but password is empty")
	}
	return nil
}

func (c SMTPConfig) validateNonNegative() error {
	if c.PoolSize < 0 {
		return errors.New("smtp: pool size must be non-negative")
	}
	if c.MaxIdleConns < 0 {
		return errors.New("smtp: max idle conns must be non-negative")
	}
	if c.MaxMessages < 0 {
		return errors.New("smtp: max messages must be non-negative")
	}
	if c.RetryDelay < 0 {
		return errors.New("smtp: retry delay must be non-negative")
	}
	if c.RetryBackoff < 0 {
		return errors.New("smtp: retry backoff must be non-negative")
	}
	return nil
}

// SMTPSender sends emails via SMTP
type SMTPSender struct {
	config  SMTPConfig
	logger  Logger
	limiter *rate.Limiter
	pool    *smtpPool // nil when pooling is disabled (PoolSize=0)
}

// NewSMTPSender creates a new SMTP sender.
// It validates the config and returns an error if it is invalid.
func NewSMTPSender(config SMTPConfig) (*SMTPSender, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = DefaultMaxRetries
	} else if config.MaxRetries < 0 {
		config.MaxRetries = 0
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = DefaultRetryDelay
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = DefaultRetryBackoff
	}
	if config.RateLimit == 0 {
		config.RateLimit = DefaultRateLimit
	}

	// Set logger
	logger := config.Logger
	if logger == nil {
		logger = NoOpLogger{}
	}

	// Create rate limiter
	var limiter *rate.Limiter
	if config.RateLimit > 0 {
		limiter = rate.NewLimiter(rate.Every(time.Second/time.Duration(config.RateLimit)), config.RateLimit)
	}

	s := &SMTPSender{
		config:  config,
		logger:  logger,
		limiter: limiter,
	}

	if config.PoolSize > 0 {
		s.pool = newSMTPPool(config, logger)
		logger.Info("connection pool enabled",
			"pool_size", config.PoolSize,
			"max_idle", s.pool.maxIdleCount,
			"max_lifetime", s.pool.maxLife.String(),
		)
	}

	return s, nil
}

// Send sends an email via SMTP with retry logic
func (s *SMTPSender) Send(ctx context.Context, email *Email) error {
	e := *email // shallow copy to avoid mutating the caller's email
	if e.From == "" {
		e.From = s.config.From
	}

	if err := e.Validate(); err != nil {
		return &Error{Op: "validate", From: e.From, To: e.To, Err: err}
	}

	if err := s.waitForRateLimit(ctx, &e); err != nil {
		return err
	}

	return s.sendWithRetries(ctx, &e)
}

// waitForRateLimit blocks until the rate limiter allows the send.
func (s *SMTPSender) waitForRateLimit(ctx context.Context, email *Email) error {
	if s.limiter == nil {
		return nil
	}
	if err := s.limiter.Wait(ctx); err != nil {
		s.logger.Error("rate limit error", "error", err)
		return &Error{Op: "rate_limit", From: email.From, To: email.To, Err: err}
	}
	return nil
}

// sendWithRetries attempts to send the email with exponential backoff retries.
func (s *SMTPSender) sendWithRetries(ctx context.Context, email *Email) error {
	var lastErr error
	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return &Error{Op: "send", From: email.From, To: email.To, Err: err}
		}

		if attempt > 0 {
			if err := s.waitForRetry(ctx, email, attempt); err != nil {
				return err
			}
		}

		lastErr = s.sendOnce(ctx, email)
		if lastErr == nil {
			// Logged at Debug so WithLogging middleware owns the user-facing
			// success log; this line is only useful for retry/attempt tracing.
			s.logger.Debug("smtp send attempt succeeded",
				"to", email.To, "subject", email.Subject, "attempt", attempt+1)
			return nil
		}

		if !isRetryableError(lastErr) {
			// Ensure callers always see a *Error from Send, regardless of
			// whether the underlying failure came from validation, message
			// building, or a context-cancelled I/O call.
			var emailErr *Error
			if errors.As(lastErr, &emailErr) {
				return lastErr
			}
			return &Error{Op: "send", From: email.From, To: email.To, Err: lastErr}
		}

		s.logger.Warn("email send attempt failed",
			"attempt", attempt+1, "error", lastErr, "to", email.To)
	}

	s.logger.Error("email send failed after all retries",
		"attempts", s.config.MaxRetries+1, "error", lastErr, "to", email.To)

	return &Error{
		Op:   "send",
		From: email.From,
		To:   email.To,
		Err:  fmt.Errorf("failed after %d attempts: %w", s.config.MaxRetries+1, lastErr),
	}
}

// waitForRetry sleeps with exponential backoff, respecting context cancellation.
func (s *SMTPSender) waitForRetry(ctx context.Context, email *Email, attempt int) error {
	const maxRetryDelay = 5 * time.Minute
	delay := time.Duration(float64(s.config.RetryDelay) *
		math.Pow(s.config.RetryBackoff, float64(attempt-1)))
	if delay > maxRetryDelay || delay <= 0 {
		delay = maxRetryDelay
	}

	s.logger.Warn("retrying email send",
		"attempt", attempt, "max_retries", s.config.MaxRetries,
		"delay", delay.String(), "to", email.To)

	timer := time.NewTimer(delay)
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		timer.Stop()
		return &Error{Op: "send", From: email.From, To: email.To, Err: ctx.Err()}
	}
}

// isRetryableError returns false for errors that are deterministic and will
// never succeed on retry (validation failures, message-building failures,
// context cancellation/deadline, and permanent SMTP 5xx replies).
func isRetryableError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// RFC 5321 §4.2.1: 5yz replies indicate permanent failure. Retrying
	// "550 mailbox unavailable" or "553 invalid sender" just burns attempts
	// and can amplify bounces on large lists.
	var tpErr *textproto.Error
	if errors.As(err, &tpErr) && tpErr.Code >= 500 && tpErr.Code < 600 {
		return false
	}
	var emailErr *Error
	if !errors.As(err, &emailErr) {
		return true
	}
	if emailErr.Op == "validate" || emailErr.Op == "build_message" {
		return false
	}
	return !errors.Is(emailErr.Err, ErrNoRecipients) &&
		!errors.Is(emailErr.Err, ErrNoSender) &&
		!errors.Is(emailErr.Err, ErrNoSubject) &&
		!errors.Is(emailErr.Err, ErrNoBody)
}

// sendOnce attempts to send an email once (no retries)
func (s *SMTPSender) sendOnce(ctx context.Context, email *Email) error {
	// Build message
	message, err := s.buildMessage(email)
	if err != nil {
		return &Error{
			Op:   "build_message",
			From: email.From,
			To:   email.To,
			Err:  err,
		}
	}

	// Extract bare email addresses for SMTP envelope commands.
	// SMTP MAIL FROM / RCPT TO require bare addresses (no display names).
	envFrom := extractAddress(email.From)
	envRecipients := make([]string, 0, len(email.To)+len(email.Cc)+len(email.Bcc))
	envRecipients = append(envRecipients, extractAddresses(email.To)...)
	envRecipients = append(envRecipients, extractAddresses(email.Cc)...)
	envRecipients = append(envRecipients, extractAddresses(email.Bcc)...)

	// Use pooled path if pool is enabled
	if s.pool != nil {
		return s.sendPooled(ctx, envFrom, envRecipients, message)
	}

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.logger.Debug("connecting to SMTP server",
		"host", s.config.Host,
		"port", s.config.Port,
		"tls", s.config.UseTLS,
	)

	// Setup authentication
	var auth smtp.Auth
	if s.config.Username != "" && s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	return s.sendDirect(ctx, addr, auth, envFrom, envRecipients, message)
}

// sendPooled sends an email using a pooled connection.
func (s *SMTPSender) sendPooled(ctx context.Context, from string, recipients []string, msg []byte) error {
	pc, err := s.pool.get(ctx)
	if err != nil {
		return fmt.Errorf("pool get: %w", err)
	}

	if err = s.sendOnConn(pc, from, recipients, msg); err != nil {
		s.pool.discard(pc)
		return err
	}

	pc.msgCount++
	s.pool.put(pc)
	return nil
}

// sendOnConn sends MAIL/RCPT/DATA on an existing authenticated connection.
// It applies the configured Timeout to the underlying conn so a stalled
// server cannot hang the send indefinitely; the deadline is cleared on
// success so the connection can sit idle in the pool.
func (s *SMTPSender) sendOnConn(pc *pooledConn, from string, recipients []string, msg []byte) error {
	if err := pc.conn.SetDeadline(time.Now().Add(s.config.Timeout)); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}
	if err := sendMailData(pc.client, from, recipients, msg); err != nil {
		_ = pc.client.Reset() // best-effort state cleanup
		return err
	}
	if err := pc.conn.SetDeadline(time.Time{}); err != nil {
		return fmt.Errorf("clear deadline: %w", err)
	}
	return nil
}

// sendDirect sends an email by dialing a new connection, optionally upgrading
// to TLS, authenticating, and delivering the message.
func (s *SMTPSender) sendDirect(ctx context.Context, addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	dialer := &net.Dialer{
		Timeout: s.config.Timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	// Bound the total time spent on subsequent SMTP commands so a slow or
	// unresponsive server cannot hang Mail/Rcpt/Data/Quit indefinitely.
	if deadlineErr := conn.SetDeadline(time.Now().Add(s.config.Timeout)); deadlineErr != nil {
		conn.Close() //nolint:errcheck // best-effort cleanup
		return fmt.Errorf("set deadline: %w", deadlineErr)
	}

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		conn.Close() //nolint:errcheck // best-effort cleanup
		return fmt.Errorf("create client: %w", err)
	}
	// client.Close also closes the underlying conn, so a single defer
	// suffices for all error paths. After a successful Quit the deferred
	// Close is a harmless no-op.
	defer client.Close() //nolint:errcheck // best-effort cleanup

	if s.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: s.config.Host,
		}
		if tlsErr := client.StartTLS(tlsConfig); tlsErr != nil {
			return fmt.Errorf("start tls: %w", tlsErr)
		}
		s.logger.Debug("TLS connection established")
	}

	if auth != nil {
		if authErr := client.Auth(auth); authErr != nil {
			return fmt.Errorf("auth: %w", authErr)
		}
		s.logger.Debug("authentication successful")
	}

	if err := sendMailData(client, from, to, msg); err != nil {
		return err
	}

	return client.Quit()
}

// sendMailData sends MAIL FROM, RCPT TO, and DATA commands on an SMTP client.
func sendMailData(client *smtp.Client, from string, to []string, msg []byte) error {
	if mailErr := client.Mail(from); mailErr != nil {
		return fmt.Errorf("set sender: %w", mailErr)
	}

	for _, recipient := range to {
		if rcptErr := client.Rcpt(recipient); rcptErr != nil {
			return fmt.Errorf("add recipient %s: %w", recipient, rcptErr)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("start data: %w", err)
	}

	if _, err = w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("write data: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	return nil
}

// buildMessage builds the email message with proper MIME encoding.
// If DKIM is configured, the message is signed before returning.
func (s *SMTPSender) buildMessage(email *Email) ([]byte, error) {
	msg, err := buildRawMessage(email)
	if err != nil {
		return nil, err
	}

	if s.config.DKIM != nil {
		msg, err = SignMessage(msg, s.config.DKIM)
		if err != nil {
			return nil, err
		}
	}

	return msg, nil
}

// Close closes the SMTP sender. If connection pooling is enabled, it shuts down
// the pool and closes all idle connections.
func (s *SMTPSender) Close() error {
	if s.pool != nil {
		return s.pool.close()
	}
	return nil
}
