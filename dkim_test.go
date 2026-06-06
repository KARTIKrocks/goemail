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
	"strings"
	"testing"
	"time"
)

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return key
}

func generateEd25519Key(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate Ed25519 key: %v", err)
	}
	return priv
}

func buildTestMessage(t *testing.T) []byte {
	t.Helper()
	e := NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetSubject("Test Subject").
		SetBody("Hello, World!")

	msg, err := buildRawMessage(e)
	if err != nil {
		t.Fatalf("failed to build message: %v", err)
	}
	return msg
}

func TestDKIMConfig_Validate(t *testing.T) {
	rsaKey := generateRSAKey(t)

	tests := []struct {
		name    string
		config  DKIMConfig
		wantErr string
	}{
		{
			name:    "missing domain",
			config:  DKIMConfig{Selector: "sel", PrivateKey: rsaKey},
			wantErr: "domain is required",
		},
		{
			name:    "missing selector",
			config:  DKIMConfig{Domain: "example.com", PrivateKey: rsaKey},
			wantErr: "selector is required",
		},
		{
			name:    "missing key",
			config:  DKIMConfig{Domain: "example.com", Selector: "sel"},
			wantErr: "private key is required",
		},
		{
			name: "invalid header canonicalization",
			config: DKIMConfig{
				Domain: "example.com", Selector: "sel", PrivateKey: rsaKey,
				HeaderCanonicalization: "invalid",
			},
			wantErr: "invalid header canonicalization",
		},
		{
			name: "invalid body canonicalization",
			config: DKIMConfig{
				Domain: "example.com", Selector: "sel", PrivateKey: rsaKey,
				BodyCanonicalization: "invalid",
			},
			wantErr: "invalid body canonicalization",
		},
		{
			name: "valid RSA config",
			config: DKIMConfig{
				Domain: "example.com", Selector: "sel", PrivateKey: rsaKey,
			},
		},
		{
			name: "valid Ed25519 config",
			config: DKIMConfig{
				Domain: "example.com", Selector: "sel", PrivateKey: generateEd25519Key(t),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSignMessage_RSA(t *testing.T) {
	rsaKey := generateRSAKey(t)
	msg := buildTestMessage(t)

	config := &DKIMConfig{
		Domain:     "example.com",
		Selector:   "default",
		PrivateKey: rsaKey,
	}

	signed, err := SignMessage(msg, config)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	// Verify DKIM-Signature header is prepended
	if !strings.HasPrefix(string(signed), "DKIM-Signature:") {
		t.Fatal("signed message should start with DKIM-Signature header")
	}

	// Verify required DKIM tags are present
	sigHeader := extractDKIMHeader(t, signed)
	requiredTags := []string{"v=1", "a=rsa-sha256", "d=example.com", "s=default", "bh=", "b=", "h="}
	for _, tag := range requiredTags {
		if !strings.Contains(sigHeader, tag) {
			t.Errorf("DKIM-Signature missing tag %q", tag)
		}
	}

	// Verify original message is intact after the DKIM-Signature header
	if !strings.Contains(string(signed), "Subject: Test Subject") {
		t.Error("original message content should be preserved")
	}
}

func TestSignMessage_Ed25519(t *testing.T) {
	edKey := generateEd25519Key(t)
	msg := buildTestMessage(t)

	config := &DKIMConfig{
		Domain:     "example.com",
		Selector:   "ed",
		PrivateKey: edKey,
	}

	signed, err := SignMessage(msg, config)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	sigHeader := extractDKIMHeader(t, signed)
	if !strings.Contains(sigHeader, "a=ed25519-sha256") {
		t.Error("DKIM-Signature should use ed25519-sha256 algorithm")
	}
}

func TestSignMessage_Expiration(t *testing.T) {
	rsaKey := generateRSAKey(t)
	msg := buildTestMessage(t)

	config := &DKIMConfig{
		Domain:     "example.com",
		Selector:   "default",
		PrivateKey: rsaKey,
		Expiration: 24 * time.Hour,
	}

	signed, err := SignMessage(msg, config)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	sigHeader := extractDKIMHeader(t, signed)
	if !strings.Contains(sigHeader, "x=") {
		t.Error("DKIM-Signature should contain x= expiration tag")
	}
}

func TestSignMessage_CustomHeaders(t *testing.T) {
	rsaKey := generateRSAKey(t)
	msg := buildTestMessage(t)

	config := &DKIMConfig{
		Domain:        "example.com",
		Selector:      "default",
		PrivateKey:    rsaKey,
		SignedHeaders: []string{"From", "Subject"},
	}

	signed, err := SignMessage(msg, config)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	sigHeader := extractDKIMHeader(t, signed)
	if !strings.Contains(sigHeader, "h=From:Subject") {
		t.Errorf("DKIM-Signature should contain h=From:Subject, got header: %s", sigHeader)
	}
}

func TestSignMessage_CustomHeadersAutoIncludesFrom(t *testing.T) {
	rsaKey := generateRSAKey(t)
	msg := buildTestMessage(t)

	config := &DKIMConfig{
		Domain:        "example.com",
		Selector:      "default",
		PrivateKey:    rsaKey,
		SignedHeaders: []string{"Subject", "Date"},
	}

	signed, err := SignMessage(msg, config)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	sigHeader := extractDKIMHeader(t, signed)
	if !strings.Contains(sigHeader, "h=From:Subject:Date") {
		t.Errorf("DKIM-Signature h= tag should auto-include From, got header: %s", sigHeader)
	}
}

func TestSignMessage_Canonicalization(t *testing.T) {
	rsaKey := generateRSAKey(t)
	msg := buildTestMessage(t)

	tests := []struct {
		name      string
		header    Canonicalization
		body      Canonicalization
		wantCanon string
	}{
		{"relaxed/relaxed (default)", "", "", "c=relaxed/relaxed"},
		{"simple/simple", CanonicalizationSimple, CanonicalizationSimple, "c=simple/simple"},
		{"relaxed/simple", CanonicalizationRelaxed, CanonicalizationSimple, "c=relaxed/simple"},
		{"simple/relaxed", CanonicalizationSimple, CanonicalizationRelaxed, "c=simple/relaxed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DKIMConfig{
				Domain:                 "example.com",
				Selector:               "default",
				PrivateKey:             rsaKey,
				HeaderCanonicalization: tt.header,
				BodyCanonicalization:   tt.body,
			}

			signed, err := SignMessage(msg, config)
			if err != nil {
				t.Fatalf("SignMessage failed: %v", err)
			}

			sigHeader := extractDKIMHeader(t, signed)
			if !strings.Contains(sigHeader, tt.wantCanon) {
				t.Errorf("expected %q in DKIM-Signature, got: %s", tt.wantCanon, sigHeader)
			}
		})
	}
}

func TestSignMessage_VerifyRSASignature(t *testing.T) {
	rsaKey := generateRSAKey(t)
	msg := buildTestMessage(t)

	config := &DKIMConfig{
		Domain:     "example.com",
		Selector:   "default",
		PrivateKey: rsaKey,
	}

	signed, err := SignMessage(msg, config)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	// Extract the b= tag value (the actual signature)
	sigHeader := extractDKIMHeader(t, signed)
	bValue := extractTagValue(sigHeader, "b=")
	if bValue == "" {
		t.Fatal("could not extract b= value from DKIM-Signature")
	}

	// Decode the signature
	sigBytes, err := base64.StdEncoding.DecodeString(bValue)
	if err != nil {
		t.Fatalf("failed to decode signature: %v", err)
	}

	// Reconstruct the data that was signed
	headers, body := splitMessage(msg)
	hdrs := parseHeaders(headers)

	canonBody := canonicalizeBody(body, config.bodyCanon())
	bodyHash := sha256.Sum256([]byte(canonBody))
	bodyHashB64 := base64.StdEncoding.EncodeToString(bodyHash[:])

	// Verify body hash matches bh= tag
	bhValue := extractTagValue(sigHeader, "bh=")
	if bhValue != bodyHashB64 {
		t.Errorf("body hash mismatch: bh=%q, computed=%q", bhValue, bodyHashB64)
	}

	// Reconstruct signed data
	signHeaders := config.signedHeaders()
	var presentHeaders []string
	for _, h := range signHeaders {
		if hdrs.has(h) {
			presentHeaders = append(presentHeaders, h)
		}
	}

	// Build DKIM tag list without b= value
	dkimTag := sigHeader[len("DKIM-Signature: "):]
	bIdx := strings.Index(dkimTag, "b=")
	if bIdx < 0 {
		t.Fatal("could not find b= in DKIM tag list")
	}
	tagListNoSig := dkimTag[:bIdx+2]

	var dataToSign strings.Builder
	for _, h := range presentHeaders {
		headerLine, _ := hdrs.get(h)
		dataToSign.WriteString(canonicalizeHeader(headerLine, config.headerCanon()))
		dataToSign.WriteString("\r\n")
	}
	dkimHeaderForSign := "DKIM-Signature: " + tagListNoSig
	dataToSign.WriteString(canonicalizeHeader(dkimHeaderForSign, config.headerCanon()))

	hash := sha256.Sum256([]byte(dataToSign.String()))
	err = rsa.VerifyPKCS1v15(&rsaKey.PublicKey, crypto.SHA256, hash[:], sigBytes)
	if err != nil {
		t.Fatalf("RSA signature verification failed: %v", err)
	}
}

func TestSignMessage_VerifyEd25519Signature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	msg := buildTestMessage(t)

	config := &DKIMConfig{
		Domain:     "example.com",
		Selector:   "ed",
		PrivateKey: priv,
	}

	signed, err := SignMessage(msg, config)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	// Extract signature
	sigHeader := extractDKIMHeader(t, signed)
	bValue := extractTagValue(sigHeader, "b=")
	sigBytes, err := base64.StdEncoding.DecodeString(bValue)
	if err != nil {
		t.Fatalf("failed to decode signature: %v", err)
	}

	// Reconstruct signed data
	headers, body := splitMessage(msg)
	hdrs := parseHeaders(headers)

	canonBody := canonicalizeBody(body, config.bodyCanon())
	bodyHash := sha256.Sum256([]byte(canonBody))
	bodyHashB64 := base64.StdEncoding.EncodeToString(bodyHash[:])

	signHeaders := config.signedHeaders()
	var presentHeaders []string
	for _, h := range signHeaders {
		if hdrs.has(h) {
			presentHeaders = append(presentHeaders, h)
		}
	}

	dkimTag := sigHeader[len("DKIM-Signature: "):]
	bIdx := strings.Index(dkimTag, "b=")
	tagListNoSig := dkimTag[:bIdx+2]

	var dataToSign strings.Builder
	for _, h := range presentHeaders {
		headerLine, _ := hdrs.get(h)
		dataToSign.WriteString(canonicalizeHeader(headerLine, config.headerCanon()))
		dataToSign.WriteString("\r\n")
	}
	dkimHeaderForSign := "DKIM-Signature: " + tagListNoSig
	dataToSign.WriteString(canonicalizeHeader(dkimHeaderForSign, config.headerCanon()))

	// Verify body hash
	bhValue := extractTagValue(sigHeader, "bh=")
	if bhValue != bodyHashB64 {
		t.Errorf("body hash mismatch: bh=%q, computed=%q", bhValue, bodyHashB64)
	}

	// RFC 8463 §3: ed25519-sha256 signs the SHA-256 hash of the data
	dataHash := sha256.Sum256([]byte(dataToSign.String()))
	if !ed25519.Verify(pub, dataHash[:], sigBytes) {
		t.Fatal("Ed25519 signature verification failed")
	}
}

func TestSignMessage_MissingFromHeader(t *testing.T) {
	rsaKey := generateRSAKey(t)

	// Craft a raw message without a From header
	msg := []byte("To: recipient@example.com\r\nSubject: No From\r\n\r\nBody here\r\n")

	config := &DKIMConfig{
		Domain:     "example.com",
		Selector:   "default",
		PrivateKey: rsaKey,
	}

	_, err := SignMessage(msg, config)
	if err == nil {
		t.Fatal("expected error when From header is missing")
	}
	if !strings.Contains(err.Error(), "no From header") {
		t.Errorf("expected 'no From header' error, got: %v", err)
	}
}

func TestSignMessage_DuplicateHeaders(t *testing.T) {
	rsaKey := generateRSAKey(t)

	// Message with duplicate Received headers (common in real email)
	msg := []byte("From: sender@example.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: Test\r\n" +
		"Received: from mx1.example.com\r\n" +
		"Received: from mx2.example.com\r\n" +
		"Date: Mon, 10 Mar 2026 12:00:00 +0000\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\nBody\r\n")

	config := &DKIMConfig{
		Domain:     "example.com",
		Selector:   "default",
		PrivateKey: rsaKey,
	}

	signed, err := SignMessage(msg, config)
	if err != nil {
		t.Fatalf("SignMessage with duplicate headers failed: %v", err)
	}

	if !strings.HasPrefix(string(signed), "DKIM-Signature:") {
		t.Fatal("signed message should start with DKIM-Signature")
	}
}

func TestSignMessage_FoldedHeaders(t *testing.T) {
	rsaKey := generateRSAKey(t)

	// Message with a folded Subject header
	msg := []byte("From: sender@example.com\r\n" +
		"To: recipient@example.com\r\n" +
		"Subject: This is a very long subject line that has been\r\n" +
		" folded across multiple lines\r\n" +
		"Date: Mon, 10 Mar 2026 12:00:00 +0000\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\nBody\r\n")

	config := &DKIMConfig{
		Domain:     "example.com",
		Selector:   "default",
		PrivateKey: rsaKey,
	}

	signed, err := SignMessage(msg, config)
	if err != nil {
		t.Fatalf("SignMessage with folded headers failed: %v", err)
	}

	if !strings.HasPrefix(string(signed), "DKIM-Signature:") {
		t.Fatal("signed message should start with DKIM-Signature")
	}
}

func TestParseHeaders_DuplicateHeaders(t *testing.T) {
	headerSection := "From: sender@example.com\r\n" +
		"Received: from mx1.example.com\r\n" +
		"Received: from mx2.example.com\r\n" +
		"Received: from mx3.example.com\r\n"

	hdrs := parseHeaders(headerSection)

	// Should have 3 Received headers
	if !hdrs.has("Received") {
		t.Fatal("expected Received headers to be present")
	}

	// get() should return bottom-to-top (RFC 6376 §5.4)
	val, ok := hdrs.get("Received")
	if !ok || val != "Received: from mx3.example.com" {
		t.Errorf("first get() should return bottom-most Received, got %q", val)
	}

	val, ok = hdrs.get("Received")
	if !ok || val != "Received: from mx2.example.com" {
		t.Errorf("second get() should return middle Received, got %q", val)
	}

	val, ok = hdrs.get("Received")
	if !ok || val != "Received: from mx1.example.com" {
		t.Errorf("third get() should return top-most Received, got %q", val)
	}

	// Now exhausted
	if hdrs.has("Received") {
		t.Error("Received should be exhausted after 3 get() calls")
	}
}

func TestParseHeaders_FoldedHeader(t *testing.T) {
	headerSection := "From: sender@example.com\r\n" +
		"Subject: This is a long\r\n" +
		" subject line\r\n"

	hdrs := parseHeaders(headerSection)

	val, ok := hdrs.get("Subject")
	if !ok {
		t.Fatal("expected Subject header to be present")
	}
	expected := "Subject: This is a long\r\n subject line"
	if val != expected {
		t.Errorf("folded header = %q, want %q", val, expected)
	}
}

func TestCanonicalizeHeader_Simple(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"From: sender@example.com", "From: sender@example.com"},
		{"Subject:  multiple   spaces", "Subject:  multiple   spaces"},
	}

	for _, tt := range tests {
		got := canonicalizeHeader(tt.input, CanonicalizationSimple)
		if got != tt.want {
			t.Errorf("canonicalizeHeader(%q, simple) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCanonicalizeHeader_Relaxed(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"From: sender@example.com", "from:sender@example.com"},
		{"Subject:  multiple   spaces", "subject:multiple spaces"},
		{"X-Custom:  value  with \t tabs", "x-custom:value with tabs"},
		{"Content-Type: text/plain;\r\n charset=UTF-8", "content-type:text/plain; charset=UTF-8"},
	}

	for _, tt := range tests {
		got := canonicalizeHeader(tt.input, CanonicalizationRelaxed)
		if got != tt.want {
			t.Errorf("canonicalizeHeader(%q, relaxed) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCanonicalizeBody_Simple(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty body", "", "\r\n"},
		{"single line", "Hello\r\n", "Hello\r\n"},
		{"trailing empty lines", "Hello\r\n\r\n\r\n", "Hello\r\n"},
		{"no trailing CRLF", "Hello", "Hello\r\n"},
		{"preserves spaces", "Hello  World\r\n", "Hello  World\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalizeBody(tt.input, CanonicalizationSimple)
			if got != tt.want {
				t.Errorf("canonicalizeBody(%q, simple) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCanonicalizeBody_Relaxed(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty body", "", "\r\n"},
		{"trailing spaces removed", "Hello  \r\n", "Hello\r\n"},
		{"internal spaces collapsed", "Hello   World\r\n", "Hello World\r\n"},
		{"tabs collapsed", "Hello\t\tWorld\r\n", "Hello World\r\n"},
		{"trailing empty lines removed", "Hello\r\n\r\n\r\n", "Hello\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalizeBody(tt.input, CanonicalizationRelaxed)
			if got != tt.want {
				t.Errorf("canonicalizeBody(%q, relaxed) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDKIMPrivateKey_RSA_PKCS1(t *testing.T) {
	rsaKey := generateRSAKey(t)
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
	})

	key, err := ParseDKIMPrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("ParseDKIMPrivateKey failed: %v", err)
	}

	if _, ok := key.(*rsa.PrivateKey); !ok {
		t.Fatalf("expected *rsa.PrivateKey, got %T", key)
	}
}

func TestParseDKIMPrivateKey_RSA_PKCS8(t *testing.T) {
	rsaKey := generateRSAKey(t)
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		t.Fatalf("failed to marshal PKCS8: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	})

	key, err := ParseDKIMPrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("ParseDKIMPrivateKey failed: %v", err)
	}

	if _, ok := key.(*rsa.PrivateKey); !ok {
		t.Fatalf("expected *rsa.PrivateKey, got %T", key)
	}
}

func TestParseDKIMPrivateKey_Ed25519(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("failed to marshal PKCS8: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	})

	key, err := ParseDKIMPrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("ParseDKIMPrivateKey failed: %v", err)
	}

	if _, ok := key.(ed25519.PrivateKey); !ok {
		t.Fatalf("expected ed25519.PrivateKey, got %T", key)
	}
}

func TestParseDKIMPrivateKey_InvalidPEM(t *testing.T) {
	_, err := ParseDKIMPrivateKey([]byte("not a PEM block"))
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestBuildRawMessage_WithDKIM(t *testing.T) {
	rsaKey := generateRSAKey(t)
	e := NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetSubject("DKIM Test").
		SetBody("Test body")

	dkim := &DKIMConfig{
		Domain:     "example.com",
		Selector:   "default",
		PrivateKey: rsaKey,
	}

	msg, err := BuildRawMessageWithDKIM(e, dkim)
	if err != nil {
		t.Fatalf("BuildRawMessageWithDKIM failed: %v", err)
	}

	if !strings.HasPrefix(string(msg), "DKIM-Signature:") {
		t.Fatal("message should start with DKIM-Signature header")
	}
}

func TestBuildRawMessage_WithoutDKIM(t *testing.T) {
	e := NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetSubject("No DKIM").
		SetBody("Test body")

	msg, err := BuildRawMessage(e)
	if err != nil {
		t.Fatalf("BuildRawMessage failed: %v", err)
	}

	if strings.HasPrefix(string(msg), "DKIM-Signature:") {
		t.Fatal("message should not have DKIM-Signature when no config provided")
	}
}

// --- Helpers ---

func extractDKIMHeader(t *testing.T, msg []byte) string {
	t.Helper()
	s := string(msg)
	// DKIM-Signature header may span multiple lines (folded)
	idx := strings.Index(s, "DKIM-Signature:")
	if idx < 0 {
		t.Fatal("DKIM-Signature header not found")
	}

	// Find the end of the DKIM-Signature header (next header that doesn't start with whitespace)
	lines := strings.Split(s[idx:], "\r\n")
	var headerLines []string
	for i, line := range lines {
		if i == 0 {
			headerLines = append(headerLines, line)
			continue
		}
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			headerLines = append(headerLines, strings.TrimSpace(line))
		} else {
			break
		}
	}

	return strings.Join(headerLines, "")
}

func extractTagValue(header, tag string) string {
	// Search for the tag at a proper boundary: start of string or after "; "
	search := header
	for {
		idx := strings.Index(search, tag)
		if idx < 0 {
			return ""
		}
		// Accept if at start of string or preceded by "; " (tag boundary)
		if idx == 0 || (idx >= 2 && search[idx-2:idx] == "; ") {
			value := search[idx+len(tag):]
			endIdx := strings.Index(value, ";")
			if endIdx >= 0 {
				value = value[:endIdx]
			}
			return strings.TrimSpace(value)
		}
		// Not at a boundary, keep searching
		search = search[idx+len(tag):]
	}
}
