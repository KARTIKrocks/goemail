# Contributing to goemail

Thank you for considering contributing to goemail! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/KARTIKrocks/goemail/issues)
2. If not, create a new issue with:
    - Clear title and description
    - Steps to reproduce
    - Expected vs actual behavior
    - Go version and OS
    - Code sample if applicable

### Suggesting Features

1. Check [Issues](https://github.com/KARTIKrocks/goemail/issues) for existing feature requests
2. Create a new issue with:
    - Clear description of the feature
    - Use cases and benefits
    - Possible implementation approach (optional)

### Submitting Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes following our coding standards
4. Add/update tests for your changes
5. Ensure all tests pass: `go test ./...`
6. Update documentation if needed
7. Commit with clear messages: `git commit -m 'Add amazing feature'`
8. Push to your fork: `git push origin feature/amazing-feature`
9. Open a Pull Request

## Development Setup

```bash
# Clone your fork
git clone https://github.com/KARTIKrocks/goemail.git
cd goemail

# Install dependencies
go mod download

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...
```

## Coding Standards

### Code Style

- Follow standard Go formatting: `gofmt -s -w .`
- Use `go vet` to check for common mistakes
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Keep functions focused and small
- Use meaningful variable and function names

### Documentation

- Add godoc comments for all exported types, functions, and constants
- Include examples in documentation where helpful
- Update README.md for user-facing changes

### Testing

- Write tests for new functionality
- Maintain or improve code coverage
- Test edge cases and error conditions
- Use table-driven tests where appropriate

Example test structure:

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Commit Messages

Use clear, descriptive commit messages:

```
Add rate limiting to SMTP sender

- Implement token bucket rate limiter
- Add RateLimit config option
- Update tests and documentation
```

## Project Structure

```
goemail/
├── doc.go              # Package documentation
├── email.go            # Core Email type, builder, Sender interface
├── mime.go             # Raw RFC 2822 message building + DKIM entry points
├── smtp.go             # SMTP sender (dial, retry, rate limit)
├── pool.go             # SMTP connection pool
├── mailer.go           # High-level Mailer facade and SendBatch
├── middleware.go       # Chainable middleware (logging, recovery, hooks)
├── async.go            # Fire-and-forget background queue
├── metrics.go          # Metrics hooks
├── webhook.go          # Provider-agnostic webhook event types
├── sanitize.go         # HTML/CSS sanitizer for safe template output
├── dkim.go             # RSA/Ed25519 DKIM signing
├── template.go         # Email templates
├── logger.go           # Logger interface + NoOpLogger
├── logger_slog.go      # slog adapter
├── mock.go             # Mock sender for testing
├── *_test.go           # Tests (same-package)
├── examples/           # Runnable examples
│   ├── basic/
│   ├── template/
│   ├── attachment/
│   ├── batch/
│   ├── middleware/
│   ├── pool/
│   └── testing/
├── providers/          # Separate submodules for HTTP/OTel adapters
│   ├── sendgrid/
│   ├── mailgun/
│   ├── ses/
│   └── otelmail/
├── README.md
├── CONTRIBUTING.md
├── LICENSE
└── go.mod
```

## Pull Request Checklist

Before submitting a PR, ensure:

- [ ] Code follows Go conventions and passes `go fmt`, `go vet`
- [ ] All tests pass: `go test ./...`
- [ ] New code has tests with good coverage
- [ ] Documentation is updated (godoc, README)
- [ ] Commit messages are clear and descriptive
- [ ] No breaking changes (or clearly documented)
- [ ] Examples updated if API changed

## Questions?

Feel free to open an issue for questions or discussions!

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
