package email

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"sort"
	"strings"
	"time"
)

// lookupHeaderFold returns the value of the first header whose key matches
// name case-insensitively. It exists so the Message-ID override works
// regardless of how the caller cased the key.
func lookupHeaderFold(headers map[string]string, name string) (string, bool) {
	if v, ok := headers[name]; ok {
		return v, true
	}
	for k, v := range headers {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	return "", false
}

// reservedHeaders lists headers that buildRawMessage writes itself.
// Custom headers with these names are skipped to avoid duplicates.
var reservedHeaders = map[string]struct{}{
	"from":                      {},
	"to":                        {},
	"cc":                        {},
	"bcc":                       {},
	"reply-to":                  {},
	"subject":                   {},
	"date":                      {},
	"message-id":                {},
	"mime-version":              {},
	"content-type":              {},
	"content-transfer-encoding": {},
}

const (
	// boundaryPrefix is the prefix for MIME boundaries
	boundaryPrefix = "boundary-"

	// altBoundaryPrefix is the prefix for alternative content boundaries
	altBoundaryPrefix = "alt-boundary-"
)

// BuildRawMessage builds a raw RFC 2822 MIME message from an Email.
// Useful for providers that accept raw messages (e.g., AWS SES).
// For DKIM-signed output, use BuildRawMessageWithDKIM.
func BuildRawMessage(e *Email) ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return buildRawMessage(e)
}

// BuildRawMessageWithDKIM builds a raw RFC 2822 MIME message and signs it
// with a DKIM-Signature header using the provided config. A nil dkim returns
// the message unsigned (equivalent to BuildRawMessage).
func BuildRawMessageWithDKIM(e *Email, dkim *DKIMConfig) ([]byte, error) {
	msg, err := BuildRawMessage(e)
	if err != nil {
		return nil, err
	}
	if dkim == nil {
		return msg, nil
	}
	return SignMessage(msg, dkim)
}

// buildRawMessage builds the email message with proper MIME encoding.
func buildRawMessage(e *Email) ([]byte, error) {
	buf := &strings.Builder{}

	// Headers
	fmt.Fprintf(buf, "From: %s\r\n", formatAddress(e.From))
	fmt.Fprintf(buf, "To: %s\r\n", formatAddressList(e.To))

	if len(e.Cc) > 0 {
		fmt.Fprintf(buf, "Cc: %s\r\n", formatAddressList(e.Cc))
	}

	if e.ReplyTo != "" {
		fmt.Fprintf(buf, "Reply-To: %s\r\n", formatAddress(e.ReplyTo))
	}

	fmt.Fprintf(buf, "Subject: %s\r\n", encodeHeader(e.Subject))

	// Message-ID: honor a user-supplied value (case-insensitive lookup),
	// otherwise generate one using the sender's domain per RFC 5322.
	if userMsgID, ok := lookupHeaderFold(e.Headers, "Message-ID"); ok {
		fmt.Fprintf(buf, "Message-ID: %s\r\n", userMsgID)
	} else {
		domain := domainFromAddress(e.From)
		msgID := fmt.Sprintf("<%s@%s>", generateUniqueID(), domain)
		fmt.Fprintf(buf, "Message-ID: %s\r\n", msgID)
	}

	// Add Date header
	fmt.Fprintf(buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))

	// Custom headers (sorted for deterministic output)
	headerKeys := make([]string, 0, len(e.Headers))
	for key := range e.Headers {
		headerKeys = append(headerKeys, key)
	}
	sort.Strings(headerKeys)
	for _, key := range headerKeys {
		if _, reserved := reservedHeaders[strings.ToLower(key)]; reserved {
			continue
		}
		fmt.Fprintf(buf, "%s: %s\r\n", key, e.Headers[key])
	}

	// MIME headers
	buf.WriteString("MIME-Version: 1.0\r\n")

	// Check if we need multipart
	hasAttachments := len(e.Attachments) > 0
	hasHTML := e.HTMLBody != ""

	switch {
	case hasAttachments:
		// multipart/mixed: body part(s) + attachments
		boundary := boundaryPrefix + generateUniqueID()
		fmt.Fprintf(buf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary)
		buf.WriteString("\r\n")

		// Body parts
		switch {
		case e.Body != "" && hasHTML:
			// Multipart alternative inside mixed
			altBoundary := altBoundaryPrefix + generateUniqueID()
			fmt.Fprintf(buf, "--%s\r\n", boundary)
			fmt.Fprintf(buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary)

			// Plain text
			fmt.Fprintf(buf, "--%s\r\n", altBoundary)
			buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			buf.WriteString(quotedPrintableEncode(e.Body))
			buf.WriteString("\r\n\r\n")

			// HTML
			fmt.Fprintf(buf, "--%s\r\n", altBoundary)
			buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			buf.WriteString(quotedPrintableEncode(e.HTMLBody))
			buf.WriteString("\r\n\r\n")
			fmt.Fprintf(buf, "--%s--\r\n", altBoundary)
		case hasHTML:
			fmt.Fprintf(buf, "--%s\r\n", boundary)
			buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			buf.WriteString(quotedPrintableEncode(e.HTMLBody))
			buf.WriteString("\r\n\r\n")
		default:
			fmt.Fprintf(buf, "--%s\r\n", boundary)
			buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
			buf.WriteString(quotedPrintableEncode(e.Body))
			buf.WriteString("\r\n\r\n")
		}

		// Attachments
		for _, att := range e.Attachments {
			fmt.Fprintf(buf, "--%s\r\n", boundary)
			fmt.Fprintf(buf, "Content-Type: %s\r\n", att.ContentType)
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			fmt.Fprintf(buf, "Content-Disposition: %s\r\n\r\n", formatDisposition(att.Filename))

			// Encode and wrap base64 at 76 characters per RFC 2045
			encoded := base64.StdEncoding.EncodeToString(att.Data)
			buf.WriteString(wrapText(encoded, 76))
			buf.WriteString("\r\n\r\n")
		}

		fmt.Fprintf(buf, "--%s--\r\n", boundary)

	case e.Body != "" && hasHTML:
		// multipart/alternative: plain text + HTML (no attachments)
		altBoundary := altBoundaryPrefix + generateUniqueID()
		fmt.Fprintf(buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary)
		buf.WriteString("\r\n")

		// Plain text
		fmt.Fprintf(buf, "--%s\r\n", altBoundary)
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(quotedPrintableEncode(e.Body))
		buf.WriteString("\r\n\r\n")

		// HTML
		fmt.Fprintf(buf, "--%s\r\n", altBoundary)
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(quotedPrintableEncode(e.HTMLBody))
		buf.WriteString("\r\n\r\n")
		fmt.Fprintf(buf, "--%s--\r\n", altBoundary)

	case hasHTML:
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(quotedPrintableEncode(e.HTMLBody))
	default:
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(quotedPrintableEncode(e.Body))
	}

	return []byte(buf.String()), nil
}

// wrapText wraps text at the specified width
func wrapText(text string, width int) string {
	var result strings.Builder
	for i := 0; i < len(text); i += width {
		end := i + width
		if end > len(text) {
			end = len(text)
		}
		result.WriteString(text[i:end])
		if end < len(text) {
			result.WriteString("\r\n")
		}
	}
	return result.String()
}

// quotedPrintableEncode encodes text using quoted-printable encoding for safe email transport.
func quotedPrintableEncode(s string) string {
	var buf strings.Builder
	w := quotedprintable.NewWriter(&buf)
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return buf.String()
}

// formatAddress returns an RFC 5322 address with any non-ASCII display name
// encoded per RFC 2047. Bare addresses are returned as-is (without angle brackets)
// to preserve the most compact valid form.
func formatAddress(addr string) string {
	parsed, err := mail.ParseAddress(addr)
	if err != nil {
		return addr
	}
	if parsed.Name == "" {
		return parsed.Address
	}
	return parsed.String()
}

// formatAddressList formats a list of addresses with RFC 2047 encoding applied
// to any non-ASCII display names.
func formatAddressList(addrs []string) string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = formatAddress(a)
	}
	return strings.Join(out, ", ")
}

// encodeHeader encodes a header value using RFC 2047 if it contains non-ASCII characters.
func encodeHeader(value string) string {
	for _, r := range value {
		if r > 127 {
			return mime.QEncoding.Encode("UTF-8", value)
		}
	}
	return value
}

// filenameReplacer removes characters that could cause header injection in
// Content-Disposition filenames.
var filenameReplacer = strings.NewReplacer(
	"\"", "_",
	"\r", "",
	"\n", "",
	"\x00", "",
	"/", "_",
	"\\", "_",
)

// sanitizeFilename removes characters that could cause header injection in
// Content-Disposition filenames.
func sanitizeFilename(name string) string {
	return filenameReplacer.Replace(name)
}

// isASCII reports whether s contains only 7-bit ASCII characters.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// formatDisposition builds the Content-Disposition header value for an
// attachment. Non-ASCII filenames are encoded per RFC 2231 (filename*=) via
// mime.FormatMediaType so strict clients can decode them; ASCII filenames
// keep the simpler filename="..." form for broader compatibility.
func formatDisposition(name string) string {
	clean := sanitizeFilename(name)
	if isASCII(clean) {
		return fmt.Sprintf(`attachment; filename="%s"`, clean)
	}
	return mime.FormatMediaType("attachment", map[string]string{"filename": clean})
}

// domainFromAddress extracts the domain part from an email address.
// It handles both bare addresses ("user@example.com") and display-name
// forms ("User <user@example.com>"). Falls back to "localhost" if
// the address cannot be parsed.
func domainFromAddress(addr string) string {
	bare := extractAddress(addr)
	if idx := strings.LastIndex(bare, "@"); idx >= 0 {
		return bare[idx+1:]
	}
	return "localhost"
}

// generateUniqueID generates a unique identifier for Message-ID and boundaries
func generateUniqueID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback: should never happen with crypto/rand
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
