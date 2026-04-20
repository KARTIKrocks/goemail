package email

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Canonicalization represents a DKIM canonicalization algorithm.
type Canonicalization string

const (
	// CanonicalizationSimple is the "simple" canonicalization algorithm (RFC 6376 §3.4.1/§3.4.3).
	CanonicalizationSimple Canonicalization = "simple"

	// CanonicalizationRelaxed is the "relaxed" canonicalization algorithm (RFC 6376 §3.4.2/§3.4.4).
	CanonicalizationRelaxed Canonicalization = "relaxed"
)

// Default headers to sign if none are specified.
// Per RFC 6376 §5.4, From is required and these are recommended.
var defaultSignedHeaders = []string{
	"From",
	"To",
	"Cc",
	"Subject",
	"Date",
	"MIME-Version",
	"Content-Type",
	"Content-Transfer-Encoding",
	"Reply-To",
	"Message-ID",
}

// DKIMConfig holds the configuration for DKIM signing.
type DKIMConfig struct {
	// Domain is the signing domain (d= tag). Required.
	Domain string

	// Selector is the DNS selector (s= tag). Required.
	Selector string

	// PrivateKey is the signer. Must be *rsa.PrivateKey or ed25519.PrivateKey.
	// Use ParseDKIMPrivateKey to load from PEM data.
	PrivateKey crypto.Signer

	// HeaderCanonicalization is the canonicalization algorithm for headers.
	// Default: CanonicalizationRelaxed.
	HeaderCanonicalization Canonicalization

	// BodyCanonicalization is the canonicalization algorithm for the body.
	// Default: CanonicalizationRelaxed.
	BodyCanonicalization Canonicalization

	// SignedHeaders lists the header field names to sign.
	// If empty, a sensible default set is used. "From" is always included.
	SignedHeaders []string

	// Expiration is the signature validity duration (x= tag).
	// Zero means no expiration.
	Expiration time.Duration
}

// Validate checks that the DKIM configuration is valid.
func (c *DKIMConfig) Validate() error {
	if c.Domain == "" {
		return errors.New("dkim: domain is required")
	}
	if c.Selector == "" {
		return errors.New("dkim: selector is required")
	}
	if c.PrivateKey == nil {
		return errors.New("dkim: private key is required")
	}

	switch c.PrivateKey.(type) {
	case *rsa.PrivateKey, ed25519.PrivateKey:
		// OK
	default:
		return fmt.Errorf("dkim: unsupported key type %T (must be *rsa.PrivateKey or ed25519.PrivateKey)", c.PrivateKey)
	}

	if c.HeaderCanonicalization != "" && c.HeaderCanonicalization != CanonicalizationSimple && c.HeaderCanonicalization != CanonicalizationRelaxed {
		return fmt.Errorf("dkim: invalid header canonicalization %q", c.HeaderCanonicalization)
	}
	if c.BodyCanonicalization != "" && c.BodyCanonicalization != CanonicalizationSimple && c.BodyCanonicalization != CanonicalizationRelaxed {
		return fmt.Errorf("dkim: invalid body canonicalization %q", c.BodyCanonicalization)
	}

	return nil
}

func (c *DKIMConfig) headerCanon() Canonicalization {
	if c.HeaderCanonicalization == "" {
		return CanonicalizationRelaxed
	}
	return c.HeaderCanonicalization
}

func (c *DKIMConfig) bodyCanon() Canonicalization {
	if c.BodyCanonicalization == "" {
		return CanonicalizationRelaxed
	}
	return c.BodyCanonicalization
}

func (c *DKIMConfig) signedHeaders() []string {
	if len(c.SignedHeaders) > 0 {
		// Ensure From is always included
		hasFrom := false
		for _, h := range c.SignedHeaders {
			if strings.EqualFold(h, "From") {
				hasFrom = true
				break
			}
		}
		// Return a copy to prevent mutation of the caller's slice
		result := make([]string, 0, len(c.SignedHeaders)+1)
		if !hasFrom {
			result = append(result, "From")
		}
		result = append(result, c.SignedHeaders...)
		return result
	}
	// Return a copy of defaults to prevent mutation of the package-level slice
	result := make([]string, len(defaultSignedHeaders))
	copy(result, defaultSignedHeaders)
	return result
}

func (c *DKIMConfig) algorithm() string {
	switch c.PrivateKey.(type) {
	case ed25519.PrivateKey:
		return "ed25519-sha256"
	default:
		return "rsa-sha256"
	}
}

// ParseDKIMPrivateKey parses a PEM-encoded private key (RSA or Ed25519) for DKIM signing.
func ParseDKIMPrivateKey(pemData []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("dkim: failed to decode PEM block")
	}

	// Try PKCS8 first (works for both RSA and Ed25519)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		signer, ok := key.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("dkim: parsed key type %T does not implement crypto.Signer", key)
		}
		return signer, nil
	}

	// Try PKCS1 (RSA only)
	rsaKey, rsaErr := x509.ParsePKCS1PrivateKey(block.Bytes)
	if rsaErr == nil {
		return rsaKey, nil
	}

	return nil, fmt.Errorf("dkim: unable to parse private key (pkcs8: %v, pkcs1: %v)", err, rsaErr)
}

// SignMessage signs a raw RFC 2822 message with a DKIM-Signature header.
// It returns a new message with the DKIM-Signature prepended.
func SignMessage(msg []byte, config *DKIMConfig) ([]byte, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Split message into headers and body
	headers, body := splitMessage(msg)

	// Canonicalize and hash the body
	canonBody := canonicalizeBody(body, config.bodyCanon())
	bodyHash := sha256.Sum256([]byte(canonBody))
	bodyHashB64 := base64.StdEncoding.EncodeToString(bodyHash[:])

	// Determine which headers are present and should be signed
	hdrs := parseHeaders(headers)
	signHeaders := config.signedHeaders()

	// RFC 6376 §5.4: From header MUST be signed
	if !hdrs.has("From") {
		return nil, errors.New("dkim: message has no From header (required by RFC 6376)")
	}

	var presentHeaders []string
	for _, h := range signHeaders {
		if hdrs.has(h) {
			presentHeaders = append(presentHeaders, h)
		}
	}

	// Build the DKIM-Signature header value (without the b= signature)
	now := time.Now()
	dkimTag := buildDKIMTagList(config, presentHeaders, bodyHashB64, now)

	// Canonicalize the signed headers (consuming bottom-to-top for duplicates per RFC 6376 §5.4).
	// If an entry in h= has no corresponding header (oversigning per §5.4.2), the
	// "null string" contributes nothing to the hashed data.
	var dataToSign strings.Builder
	for _, h := range presentHeaders {
		headerLine, ok := hdrs.get(h)
		if !ok {
			continue
		}
		dataToSign.WriteString(canonicalizeHeader(headerLine, config.headerCanon()))
		dataToSign.WriteString("\r\n")
	}

	// Canonicalize and append the DKIM-Signature header itself (without trailing \r\n)
	dkimHeader := "DKIM-Signature: " + dkimTag
	dataToSign.WriteString(canonicalizeHeader(dkimHeader, config.headerCanon()))

	// Sign the data
	signature, err := signData([]byte(dataToSign.String()), config)
	if err != nil {
		return nil, fmt.Errorf("dkim: signing failed: %w", err)
	}

	sigB64 := base64.StdEncoding.EncodeToString(signature)

	// Build the complete DKIM-Signature header with folded signature
	fullDKIMHeader := "DKIM-Signature: " + dkimTag + foldSignature(sigB64) + "\r\n"

	// Prepend the DKIM-Signature to the original message
	result := make([]byte, 0, len(fullDKIMHeader)+len(msg))
	result = append(result, []byte(fullDKIMHeader)...)
	result = append(result, msg...)

	return result, nil
}

// splitMessage splits a raw message into header section and body.
// The separator is the first occurrence of \r\n\r\n.
func splitMessage(msg []byte) (string, string) {
	s := string(msg)
	idx := strings.Index(s, "\r\n\r\n")
	if idx < 0 {
		return s, ""
	}
	return s[:idx+2], s[idx+4:] // headers include trailing \r\n, body starts after separator
}

// parsedHeaders holds parsed header data preserving order and duplicates.
type parsedHeaders struct {
	// byName maps lowercase header name to all occurrences in message order.
	byName map[string][]string
}

// get returns the next unseen occurrence of a header, from bottom to top,
// per RFC 6376 §5.4 for signing duplicate headers.
func (p *parsedHeaders) get(name string) (string, bool) {
	lname := strings.ToLower(name)
	entries := p.byName[lname]
	if len(entries) == 0 {
		return "", false
	}
	// Pop from the end (bottom-to-top per RFC 6376 §5.4)
	last := entries[len(entries)-1]
	p.byName[lname] = entries[:len(entries)-1]
	return last, true
}

// has returns whether any occurrence of the header remains.
func (p *parsedHeaders) has(name string) bool {
	return len(p.byName[strings.ToLower(name)]) > 0
}

// parseHeaders parses the header section into a structure that preserves
// duplicate headers and supports bottom-to-top consumption per RFC 6376 §5.4.
func parseHeaders(headerSection string) *parsedHeaders {
	result := &parsedHeaders{byName: make(map[string][]string)}
	lines := strings.Split(headerSection, "\r\n")

	var currentHeader string
	var currentName string
	for _, line := range lines {
		if line == "" {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			// Continuation of previous header (folded)
			currentHeader += "\r\n" + line
			// Update the last entry for this header name
			entries := result.byName[currentName]
			if len(entries) > 0 {
				entries[len(entries)-1] = currentHeader
			}
		} else {
			// New header
			currentHeader = line
			colonIdx := strings.Index(line, ":")
			if colonIdx >= 0 {
				currentName = strings.ToLower(strings.TrimSpace(line[:colonIdx]))
				result.byName[currentName] = append(result.byName[currentName], line)
			}
		}
	}

	return result
}

// canonicalizeHeader applies header canonicalization per RFC 6376 §3.4.1/§3.4.2.
func canonicalizeHeader(header string, algo Canonicalization) string {
	if algo == CanonicalizationSimple {
		return header
	}

	// Relaxed header canonicalization (RFC 6376 §3.4.2):
	// 1. Convert header name to lowercase
	// 2. Unfold header (remove \r\n followed by WSP)
	// 3. Collapse WSP sequences to single space
	// 4. Remove WSP before and after the colon
	// 5. Remove trailing WSP

	before, after, ok := strings.Cut(header, ":")
	if !ok {
		return header
	}

	name := strings.ToLower(strings.TrimSpace(before))
	value := after

	// Unfold: remove CRLF followed by whitespace
	value = strings.ReplaceAll(value, "\r\n\t", " ")
	value = strings.ReplaceAll(value, "\r\n ", " ")

	// Collapse whitespace runs to a single space
	var collapsed strings.Builder
	inWSP := false
	for _, c := range value {
		if c == ' ' || c == '\t' {
			if !inWSP {
				collapsed.WriteByte(' ')
				inWSP = true
			}
		} else {
			collapsed.WriteRune(c)
			inWSP = false
		}
	}

	result := strings.TrimSpace(collapsed.String())
	return name + ":" + result
}

// canonicalizeBody applies body canonicalization per RFC 6376 §3.4.3/§3.4.4.
func canonicalizeBody(body string, algo Canonicalization) string {
	if algo == CanonicalizationSimple {
		// Simple body canonicalization (RFC 6376 §3.4.3):
		// - Ensure body ends with CRLF
		// - Remove trailing empty lines (lines with only CRLF)
		body = removeTrailingEmptyLines(body)
		if body == "" {
			return "\r\n"
		}
		if !strings.HasSuffix(body, "\r\n") {
			body += "\r\n"
		}
		return body
	}

	// Relaxed body canonicalization (RFC 6376 §3.4.4):
	// 1. Reduce WSP sequences at end of each line to empty
	// 2. Reduce all sequences of WSP within a line to single SP
	// 3. Ignore all empty lines at the end of the body
	// 4. Ensure body ends with CRLF (if non-empty)

	lines := strings.Split(body, "\r\n")
	var result []string
	for _, line := range lines {
		// Collapse internal whitespace sequences to a single space
		var collapsed strings.Builder
		inWSP := false
		for _, c := range line {
			if c == ' ' || c == '\t' {
				if !inWSP {
					collapsed.WriteByte(' ')
					inWSP = true
				}
			} else {
				collapsed.WriteRune(c)
				inWSP = false
			}
		}
		// Remove trailing whitespace from line
		result = append(result, strings.TrimRight(collapsed.String(), " \t"))
	}

	// Remove trailing empty lines
	for len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}

	if len(result) == 0 {
		return "\r\n"
	}

	return strings.Join(result, "\r\n") + "\r\n"
}

// removeTrailingEmptyLines removes trailing CRLF-only lines.
func removeTrailingEmptyLines(body string) string {
	for strings.HasSuffix(body, "\r\n\r\n") {
		body = body[:len(body)-2]
	}
	return body
}

// buildDKIMTagList builds the DKIM-Signature tag list (without the b= value).
func buildDKIMTagList(config *DKIMConfig, headers []string, bodyHashB64 string, now time.Time) string {
	headerCanon := config.headerCanon()
	bodyCanon := config.bodyCanon()

	var tags strings.Builder
	tags.WriteString("v=1; ")
	tags.WriteString("a=" + config.algorithm() + "; ")
	tags.WriteString("c=" + string(headerCanon) + "/" + string(bodyCanon) + "; ")
	tags.WriteString("d=" + config.Domain + "; ")
	tags.WriteString("s=" + config.Selector + "; ")
	tags.WriteString("t=" + fmt.Sprintf("%d", now.Unix()) + "; ")

	if config.Expiration > 0 {
		tags.WriteString("x=" + fmt.Sprintf("%d", now.Add(config.Expiration).Unix()) + "; ")
	}

	tags.WriteString("h=" + strings.Join(headers, ":") + "; ")
	tags.WriteString("bh=" + bodyHashB64 + "; ")
	tags.WriteString("b=")

	return tags.String()
}

// signData signs the data using the configured private key.
func signData(data []byte, config *DKIMConfig) ([]byte, error) {
	switch key := config.PrivateKey.(type) {
	case *rsa.PrivateKey:
		hash := sha256.Sum256(data)
		return rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	case ed25519.PrivateKey:
		// RFC 8463 §3: ed25519-sha256 signs the SHA-256 hash of the
		// canonicalized signed headers, not the raw data.
		hash := sha256.Sum256(data)
		return ed25519.Sign(key, hash[:]), nil
	default:
		return nil, fmt.Errorf("dkim: unsupported key type %T", key)
	}
}

// foldSignature folds a base64-encoded signature for insertion into the DKIM header.
// Lines are wrapped at 76 characters with leading whitespace per RFC 6376 §3.7.
func foldSignature(sig string) string {
	var result strings.Builder
	const lineLen = 72 // leave room for leading whitespace
	for i := 0; i < len(sig); i += lineLen {
		end := min(i+lineLen, len(sig))
		if i == 0 {
			result.WriteString(sig[i:end])
		} else {
			result.WriteString("\r\n        " + sig[i:end])
		}
	}
	return result.String()
}
