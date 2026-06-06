package email

import (
	"context"
	htmltemplate "html/template"
	"strconv"
	"strings"
	"unicode"
)

// Policy defines which HTML elements and attributes are allowed in sanitized
// output. Use [EmailPolicy] for a sensible default or build a custom policy
// with [NewPolicy].
type Policy struct {
	// element name -> set of allowed attribute names (lowercase)
	elements map[string]map[string]bool

	// attribute name -> set of allowed URL protocols (lowercase)
	protocols map[string]map[string]bool

	// elements whose opening and closing tags AND content are removed entirely
	stripElements map[string]bool

	// global attributes allowed on every element
	globalAttrs map[string]bool
}

// NewPolicy returns an empty policy that strips all HTML.
// Use the builder methods to allow specific elements and attributes.
func NewPolicy() *Policy {
	return &Policy{
		elements:      make(map[string]map[string]bool),
		protocols:     make(map[string]map[string]bool),
		stripElements: make(map[string]bool),
		globalAttrs:   make(map[string]bool),
	}
}

// AllowElements adds elements to the allowlist with no extra attributes
// beyond globals. If an element was already allowed, its attribute set is
// preserved.
func (p *Policy) AllowElements(elements ...string) *Policy {
	for _, el := range elements {
		el = strings.ToLower(el)
		if _, ok := p.elements[el]; !ok {
			p.elements[el] = make(map[string]bool)
		}
	}
	return p
}

// AllowAttributes adds allowed attributes for the given element.
// The element is implicitly added to the allowlist if not already present.
func (p *Policy) AllowAttributes(element string, attrs ...string) *Policy {
	element = strings.ToLower(element)
	m, ok := p.elements[element]
	if !ok {
		m = make(map[string]bool)
		p.elements[element] = m
	}
	for _, a := range attrs {
		m[strings.ToLower(a)] = true
	}
	return p
}

// AllowGlobalAttributes adds attributes that are allowed on every element.
func (p *Policy) AllowGlobalAttributes(attrs ...string) *Policy {
	for _, a := range attrs {
		p.globalAttrs[strings.ToLower(a)] = true
	}
	return p
}

// AllowURLProtocols sets the allowed URL protocols for an attribute
// (typically "href" or "src"). Values should be lowercase without a
// trailing colon (e.g. "https", "mailto").
func (p *Policy) AllowURLProtocols(attr string, protocols ...string) *Policy {
	attr = strings.ToLower(attr)
	m, ok := p.protocols[attr]
	if !ok {
		m = make(map[string]bool)
		p.protocols[attr] = m
	}
	for _, proto := range protocols {
		m[strings.ToLower(proto)] = true
	}
	return p
}

// StripElements marks elements for complete removal — both the tags and
// their inner content are discarded. This is appropriate for elements like
// <script> and <style> where the content itself is dangerous.
func (p *Policy) StripElements(elements ...string) *Policy {
	for _, el := range elements {
		p.stripElements[strings.ToLower(el)] = true
	}
	return p
}

// isElementAllowed reports whether the element is in the allowlist.
func (p *Policy) isElementAllowed(tag string) bool {
	_, ok := p.elements[strings.ToLower(tag)]
	return ok
}

// isStripElement reports whether the element should be stripped entirely.
func (p *Policy) isStripElement(tag string) bool {
	return p.stripElements[strings.ToLower(tag)]
}

// isAttrAllowed reports whether the attribute is allowed for the element.
func (p *Policy) isAttrAllowed(tag, attr string) bool {
	tag = strings.ToLower(tag)
	attr = strings.ToLower(attr)

	if p.globalAttrs[attr] {
		return true
	}
	m, ok := p.elements[tag]
	if !ok {
		return false
	}
	return m[attr]
}

// isURLSafe checks whether a URL attribute value uses an allowed protocol.
func (p *Policy) isURLSafe(attr, value string) bool {
	attr = strings.ToLower(attr)
	protos, ok := p.protocols[attr]
	if !ok {
		// No protocol restriction for this attribute.
		return true
	}

	value = strings.TrimSpace(value)

	// Decode common HTML entities that could hide protocols.
	value = decodeHTMLEntities(value)

	// Browsers strip ASCII tab, LF, CR from URLs before parsing (WHATWG URL
	// spec §4.1). Without this, "java\tscript:alert(1)" reaches the renderer
	// as "javascript:alert(1)" but the protocol check below would see a
	// non-alphabetic scheme and fall through to "allow". Also strip NUL
	// defensively.
	value = urlNormalizeReplacer.Replace(value)

	// Strip leading whitespace and control chars that some browsers ignore.
	value = strings.TrimLeftFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	})
	value = strings.ToLower(value)

	// Find the protocol.
	colonIdx := strings.Index(value, ":")
	if colonIdx < 0 {
		// Relative URL or fragment — allow.
		return true
	}

	proto := value[:colonIdx]
	// Protocols must be purely alphabetic (no spaces, tabs, etc.)
	for _, r := range proto {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			// Not a protocol prefix (could be a Windows path like C:\) — allow.
			return true
		}
	}

	return protos[proto]
}

// EmailPolicy returns a [Policy] configured with elements and attributes
// that are safe and widely supported across email clients (Gmail, Outlook,
// Apple Mail, Yahoo Mail).
func EmailPolicy() *Policy {
	p := NewPolicy()

	// Global attributes allowed on any element.
	p.AllowGlobalAttributes("style", "class", "id", "dir", "lang", "title", "align", "valign", "bgcolor")

	// Structure
	p.AllowElements("html", "head", "body", "div", "span", "p", "br", "hr",
		"table", "thead", "tbody", "tfoot", "tr", "caption", "center")

	// Table cells with layout attributes
	p.AllowAttributes("td", "colspan", "rowspan", "width", "height")
	p.AllowAttributes("th", "colspan", "rowspan", "width", "height")
	p.AllowAttributes("table", "width", "cellpadding", "cellspacing", "border")

	// Text formatting
	p.AllowElements("h1", "h2", "h3", "h4", "h5", "h6",
		"strong", "b", "em", "i", "u", "s", "strike",
		"sub", "sup", "blockquote", "pre", "code")

	// Lists
	p.AllowElements("ul", "ol", "li")

	// Links
	p.AllowAttributes("a", "href", "target", "title")
	p.AllowURLProtocols("href", "http", "https", "mailto")

	// Images
	p.AllowAttributes("img", "src", "alt", "width", "height")
	p.AllowURLProtocols("src", "http", "https", "cid")

	// Legacy font element (widely supported in email)
	p.AllowAttributes("font", "color", "size", "face")

	// Elements to strip entirely (tags + content)
	p.StripElements("script", "style", "iframe", "object", "embed",
		"form", "input", "textarea", "select", "button",
		"applet", "svg", "math", "link", "meta", "base")

	return p
}

// defaultPolicy is the cached EmailPolicy instance used by SanitizeHTML.
var defaultPolicy = EmailPolicy()

// SanitizeHTML sanitizes HTML using the default [EmailPolicy].
// It removes elements, attributes, and URL protocols that are unsafe or
// unsupported in email clients.
func SanitizeHTML(html string) string {
	return SanitizeHTMLWithPolicy(html, defaultPolicy)
}

// SanitizeHTMLWithPolicy sanitizes HTML using a custom [Policy].
func SanitizeHTMLWithPolicy(html string, p *Policy) string {
	if html == "" {
		return ""
	}

	var buf strings.Builder
	buf.Grow(len(html))
	sanitize(&buf, html, p)
	return buf.String()
}

// sanitize is the core scanner that walks through HTML, filtering tags and
// attributes according to the policy.
func sanitize(buf *strings.Builder, input string, p *Policy) {
	i := 0
	for i < len(input) {
		tagStart := strings.IndexByte(input[i:], '<')
		if tagStart < 0 {
			buf.WriteString(input[i:])
			break
		}
		tagStart += i
		buf.WriteString(input[i:tagStart])

		// Skip HTML comments.
		if strings.HasPrefix(input[tagStart:], "<!--") {
			end := strings.Index(input[tagStart+4:], "-->")
			if end < 0 {
				break
			}
			i = tagStart + 4 + end + 3
			continue
		}

		tagEnd := findTagEnd(input, tagStart)
		if tagEnd < 0 {
			buf.WriteString(escapeText(input[tagStart:]))
			break
		}

		rawTag := input[tagStart : tagEnd+1]
		i = tagEnd + 1

		tag, attrs, isClosing, selfClosing := parseTag(rawTag)
		if tag == "" {
			continue
		}

		tagLower := strings.ToLower(tag)
		i = handleTag(buf, input, i, tagLower, attrs, isClosing, selfClosing, p)
	}
}

// handleTag processes a single parsed tag, returning the updated position.
func handleTag(buf *strings.Builder, input string, pos int, tag string, attrs []attribute, isClosing, selfClosing bool, p *Policy) int {
	if p.isStripElement(tag) {
		if !isClosing && !selfClosing {
			return skipToClosingTag(input, pos, tag)
		}
		return pos
	}

	if !p.isElementAllowed(tag) {
		return pos
	}

	writeAllowedTag(buf, tag, attrs, isClosing, selfClosing, p)
	return pos
}

// writeAllowedTag rebuilds a tag with only allowed attributes.
func writeAllowedTag(buf *strings.Builder, tag string, attrs []attribute, isClosing, selfClosing bool, p *Policy) {
	buf.WriteByte('<')
	if isClosing {
		buf.WriteByte('/')
	}
	buf.WriteString(tag)

	for _, attr := range attrs {
		writeAllowedAttr(buf, tag, attr, p)
	}

	if selfClosing {
		buf.WriteString(" /")
	}
	buf.WriteByte('>')
}

// writeAllowedAttr writes a single attribute if it passes policy checks.
func writeAllowedAttr(buf *strings.Builder, tag string, attr attribute, p *Policy) {
	name := strings.ToLower(attr.name)
	if !p.isAttrAllowed(tag, name) {
		return
	}

	val := attr.value
	if name == "style" {
		val = sanitizeCSS(val)
		if val == "" {
			return
		}
	}

	if !p.isURLSafe(name, val) {
		return
	}

	buf.WriteByte(' ')
	buf.WriteString(name)
	buf.WriteString(`="`)
	buf.WriteString(escapeAttrValue(val))
	buf.WriteByte('"')
}

// attribute holds a parsed HTML attribute.
type attribute struct {
	name  string
	value string
}

// parseTag extracts the tag name, attributes, and whether it is a closing
// or self-closing tag from a raw tag string like `<div class="x">`.
func parseTag(raw string) (tag string, attrs []attribute, isClosing, selfClosing bool) {
	// Trim < and >
	s := raw[1 : len(raw)-1]

	if strings.HasSuffix(s, "/") {
		s = s[:len(s)-1]
		selfClosing = true
	}

	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "/") {
		isClosing = true
		s = strings.TrimSpace(s[1:])
	}

	// Extract tag name.
	nameEnd := strings.IndexFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == '/' || r == '>'
	})
	if nameEnd < 0 {
		tag = s
		return
	}
	tag = s[:nameEnd]
	s = strings.TrimSpace(s[nameEnd:])

	attrs = parseAttrs(s)
	return
}

// parseAttrs parses HTML attributes from the remainder of a tag after
// the tag name has been extracted.
func parseAttrs(s string) []attribute {
	var attrs []attribute
	for s = strings.TrimSpace(s); len(s) > 0; s = strings.TrimSpace(s) {
		var attr attribute
		attr, s = parseOneAttr(s)
		if attr.name != "" && attr.name != "/" {
			attrs = append(attrs, attr)
		}
	}
	return attrs
}

// parseOneAttr parses a single attribute from s and returns the attribute
// and the remaining unparsed string.
func parseOneAttr(s string) (attribute, string) {
	eqIdx := strings.IndexByte(s, '=')
	spaceIdx := strings.IndexFunc(s, unicode.IsSpace)

	// Boolean attribute (no = sign, or space comes before =).
	if eqIdx < 0 || (spaceIdx >= 0 && spaceIdx < eqIdx) {
		end := spaceIdx
		if end < 0 {
			return attribute{name: s}, ""
		}
		return attribute{name: s[:end]}, s[end:]
	}

	attrName := strings.TrimSpace(s[:eqIdx])
	s = strings.TrimSpace(s[eqIdx+1:])
	if len(s) == 0 {
		return attribute{name: attrName}, ""
	}

	val, rest := parseAttrValue(s)
	return attribute{name: attrName, value: val}, rest
}

// parseAttrValue extracts an attribute value (quoted or unquoted) from s.
func parseAttrValue(s string) (string, string) {
	if s[0] == '"' || s[0] == '\'' {
		quote := s[0]
		end := strings.IndexByte(s[1:], quote)
		if end < 0 {
			return s[1:], ""
		}
		return s[1 : end+1], s[end+2:]
	}

	end := strings.IndexFunc(s, unicode.IsSpace)
	if end < 0 {
		return s, ""
	}
	return s[:end], s[end:]
}

// findTagEnd returns the index of the closing '>' for the tag starting at pos.
// It respects quoted attribute values. Returns -1 if not found.
func findTagEnd(s string, pos int) int {
	inSingle := false
	inDouble := false
	for i := pos + 1; i < len(s); i++ {
		switch s[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '>':
			if !inSingle && !inDouble {
				return i
			}
		}
	}
	return -1
}

// skipToClosingTag advances past the matching closing tag for tagName,
// handling nested occurrences. Returns the new position.
func skipToClosingTag(s string, pos int, tagName string) int {
	closingTag := "</" + tagName
	openTag := "<" + tagName
	depth := 1

	for i := pos; i < len(s) && depth > 0; {
		next := strings.IndexByte(s[i:], '<')
		if next < 0 {
			return len(s)
		}
		next += i

		remaining := s[next:]

		if hasPrefixFold(remaining, closingTag) && isTagBoundary(remaining, len(closingTag)) {
			depth--
			if depth == 0 {
				return skipPastTagEnd(s, next)
			}
		} else if hasPrefixFold(remaining, openTag) && isTagBoundary(remaining, len(openTag)) {
			depth++
		}

		i = next + 1
	}
	return len(s)
}

// hasPrefixFold reports whether s starts with prefix, ignoring ASCII case.
func hasPrefixFold(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := range len(prefix) {
		a, b := s[i], prefix[i]
		if a == b {
			continue
		}
		// ASCII lowercase comparison
		if 'A' <= a && a <= 'Z' {
			a += 'a' - 'A'
		}
		if 'A' <= b && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

// isTagBoundary checks whether the character at position afterLen in s
// is a valid tag boundary (>, space, tab, or /), confirming the prefix
// is a complete tag name and not a prefix of a longer name (e.g. <scriptx).
func isTagBoundary(s string, afterLen int) bool {
	if afterLen >= len(s) {
		return true
	}
	ch := s[afterLen]
	return ch == '>' || ch == ' ' || ch == '\t' || ch == '/'
}

// skipPastTagEnd finds the closing '>' after pos and returns the position
// after it. Returns len(s) if no closing '>' is found.
func skipPastTagEnd(s string, pos int) int {
	end := findTagEnd(s, pos)
	if end < 0 {
		return len(s)
	}
	return end + 1
}

// sanitizeCSS removes dangerous CSS constructs from a style attribute value.
// url(...) references are preserved when their protocol is http, https, or cid
// (or when the URL is relative); any other protocol drops the entire style.
func sanitizeCSS(css string) string {
	lower := strings.ToLower(css)

	dangerous := []string{
		"expression", "javascript:", "vbscript:",
		"-moz-binding", "behavior:",
	}
	for _, d := range dangerous {
		if strings.Contains(lower, d) {
			return ""
		}
	}

	if !cssURLsSafe(lower) {
		return ""
	}
	return css
}

// cssURLsSafe returns true iff every url(...) in css uses an allowed protocol.
// Input is expected to already be lowercased.
func cssURLsSafe(css string) bool {
	allowed := map[string]bool{"http": true, "https": true, "cid": true}
	for i := 0; i < len(css); {
		idx := strings.Index(css[i:], "url(")
		if idx < 0 {
			return true
		}
		start := i + idx + len("url(")
		end := strings.IndexByte(css[start:], ')')
		if end < 0 {
			return false // unterminated url(
		}
		raw := strings.TrimSpace(css[start : start+end])
		raw = strings.Trim(raw, `"'`)
		raw = strings.TrimSpace(raw)
		colon := strings.Index(raw, ":")
		if colon >= 0 {
			proto := raw[:colon]
			for _, r := range proto {
				if r < 'a' || r > 'z' {
					return false
				}
			}
			if !allowed[proto] {
				return false
			}
		}
		i = start + end + 1
	}
	return true
}

// escapeAttrValue escapes characters that are special in HTML attribute values.
func escapeAttrValue(s string) string {
	if !strings.ContainsAny(s, `"&<>`) {
		return s
	}
	var buf strings.Builder
	buf.Grow(len(s) + 10)
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString("&quot;")
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// escapeText escapes HTML special characters in text content.
func escapeText(s string) string {
	if !strings.ContainsAny(s, "<>&") {
		return s
	}
	var buf strings.Builder
	buf.Grow(len(s) + 10)
	for _, r := range s {
		switch r {
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '&':
			buf.WriteString("&amp;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// namedEntityReplacer handles named HTML entities used in URL obfuscation.
var namedEntityReplacer = strings.NewReplacer(
	"&colon;", ":",
	"&tab;", "\t",
	"&newline;", "\n",
)

// urlNormalizeReplacer removes characters that browsers strip from URLs
// before parsing (WHATWG URL spec §4.1: ASCII tab, LF, CR). NUL is stripped
// defensively to match common C-string truncation in URL handlers.
var urlNormalizeReplacer = strings.NewReplacer(
	"\t", "",
	"\n", "",
	"\r", "",
	"\x00", "",
)

// decodeHTMLEntities decodes HTML character references (numeric and named)
// that are commonly used to obfuscate URL protocols in XSS attacks.
func decodeHTMLEntities(s string) string {
	s = namedEntityReplacer.Replace(s)

	var buf strings.Builder
	buf.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '&' && i+3 < len(s) && s[i+1] == '#' {
			semiIdx := strings.IndexByte(s[i+2:], ';')
			if semiIdx > 0 && semiIdx < 10 {
				if ch, ok := decodeNumericRef(s[i+2 : i+2+semiIdx]); ok {
					if ch != 0 {
						buf.WriteRune(ch)
					}
					i = i + 2 + semiIdx + 1
					continue
				}
			}
		}
		buf.WriteByte(s[i])
		i++
	}
	return buf.String()
}

// decodeNumericRef decodes a numeric character reference (the part between &# and ;).
// Supports decimal (&#NNN;) and hex (&#xHH; / &#XHH;) forms.
func decodeNumericRef(ref string) (rune, bool) {
	if len(ref) == 0 {
		return 0, false
	}

	var n int64
	var err error
	if ref[0] == 'x' || ref[0] == 'X' {
		if len(ref) < 2 {
			return 0, false
		}
		n, err = strconv.ParseInt(ref[1:], 16, 32)
	} else {
		n, err = strconv.ParseInt(ref, 10, 32)
	}

	if err != nil || n < 0 || n > unicode.MaxRune {
		return 0, false
	}
	return rune(n), true
}

// --- Template integration ---

// SanitizeFuncMap returns a [htmltemplate.FuncMap] with a "sanitize" function
// that sanitizes HTML using the default [EmailPolicy].
//
// Usage:
//
//	tmpl := template.New("email").Funcs(email.SanitizeFuncMap())
//
// In templates:
//
//	{{.UserContent | sanitize}}
func SanitizeFuncMap() htmltemplate.FuncMap {
	return SanitizeFuncMapWithPolicy(defaultPolicy)
}

// SanitizeFuncMapWithPolicy returns a [htmltemplate.FuncMap] with a "sanitize"
// function that uses a custom [Policy].
func SanitizeFuncMapWithPolicy(p *Policy) htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"sanitize": func(html string) htmltemplate.HTML {
			return htmltemplate.HTML(SanitizeHTMLWithPolicy(html, p))
		},
	}
}

// --- Middleware ---

// WithSanitization returns a [Middleware] that sanitizes the HTMLBody of
// every email using the default [EmailPolicy] before sending.
// This acts as a safety net so that even if template authors forget to
// sanitize user content, the email body is cleaned before delivery.
func WithSanitization() Middleware {
	return WithSanitizationPolicy(defaultPolicy)
}

// WithSanitizationPolicy returns a [Middleware] that sanitizes the HTMLBody
// using a custom [Policy].
func WithSanitizationPolicy(p *Policy) Middleware {
	return func(next Sender) Sender {
		return &sanitizingSender{next: next, policy: p}
	}
}

type sanitizingSender struct {
	next   Sender
	policy *Policy
}

func (s *sanitizingSender) Send(ctx context.Context, e *Email) error {
	if e.HTMLBody != "" {
		cp := *e // shallow copy to avoid mutating the caller's email
		cp.HTMLBody = SanitizeHTMLWithPolicy(e.HTMLBody, s.policy)
		e = &cp
	}
	return s.next.Send(ctx, e)
}

func (s *sanitizingSender) Close() error {
	return s.next.Close()
}
