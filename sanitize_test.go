package email

import (
	"context"
	htmltemplate "html/template"
	"strings"
	"testing"
)

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // substrings that must appear
		excludes []string // substrings that must NOT appear
	}{
		{
			name:     "empty input",
			input:    "",
			contains: nil,
		},
		{
			name:     "plain text passthrough",
			input:    "Hello World",
			contains: []string{"Hello World"},
		},
		{
			name:     "allowed elements preserved",
			input:    `<p>Hello</p><br/><strong>bold</strong>`,
			contains: []string{"<p>", "</p>", "<br />", "<strong>", "bold"},
		},
		{
			name:     "script tag stripped with content",
			input:    `<p>Hello</p><script>alert('xss')</script><p>World</p>`,
			contains: []string{"<p>Hello</p>", "<p>World</p>"},
			excludes: []string{"<script", "alert", "xss"},
		},
		{
			name:     "style tag stripped with content",
			input:    `<div>Text</div><style>.x { color: red; }</style>`,
			contains: []string{"<div>Text</div>"},
			excludes: []string{"<style", "color: red"},
		},
		{
			name:     "iframe removed",
			input:    `<p>Before</p><iframe src="evil.com"></iframe><p>After</p>`,
			contains: []string{"<p>Before</p>", "<p>After</p>"},
			excludes: []string{"<iframe", "evil.com"},
		},
		{
			name:     "form elements stripped",
			input:    `<form action="/steal"><input type="text"><button>Submit</button></form>`,
			excludes: []string{"<form", "<input", "<button", "/steal"},
		},
		{
			name:     "unknown tag removed but content preserved",
			input:    `<blink>Important!</blink>`,
			contains: []string{"Important!"},
			excludes: []string{"<blink", "</blink"},
		},
		{
			name:     "event handler attributes stripped",
			input:    `<p onclick="alert(1)" onmouseover="steal()">Text</p>`,
			contains: []string{"<p>", "Text"},
			excludes: []string{"onclick", "onmouseover", "alert", "steal"},
		},
		{
			name:     "javascript: URL blocked in href",
			input:    `<a href="javascript:alert(1)">Click</a>`,
			contains: []string{"<a>", "Click"},
			excludes: []string{"javascript"},
		},
		{
			name:     "safe href preserved",
			input:    `<a href="https://example.com">Link</a>`,
			contains: []string{`href="https://example.com"`, "Link"},
		},
		{
			name:     "mailto href preserved",
			input:    `<a href="mailto:user@example.com">Email</a>`,
			contains: []string{`href="mailto:user@example.com"`},
		},
		{
			name:     "data: URL blocked in img src",
			input:    `<img src="data:image/svg+xml,<svg onload=alert(1)>" alt="x">`,
			contains: []string{"<img", `alt="x"`},
			excludes: []string{"data:"},
		},
		{
			name:     "cid: URL allowed in img src",
			input:    `<img src="cid:logo123" alt="Logo">`,
			contains: []string{`src="cid:logo123"`, `alt="Logo"`},
		},
		{
			name:     "style attribute preserved when safe",
			input:    `<p style="color: red; font-size: 14px;">Styled</p>`,
			contains: []string{`style="color: red; font-size: 14px;"`, "Styled"},
		},
		{
			name:     "style attribute with expression removed",
			input:    `<p style="width: expression(alert(1))">Text</p>`,
			contains: []string{"<p>", "Text"},
			excludes: []string{"expression", "alert"},
		},
		{
			name:     "style attribute with javascript removed",
			input:    `<div style="background: url(javascript:alert(1))">X</div>`,
			contains: []string{"<div>", "X"},
			excludes: []string{"javascript"},
		},
		{
			name:     "table with attributes",
			input:    `<table width="600" cellpadding="0" border="1"><tr><td colspan="2">Cell</td></tr></table>`,
			contains: []string{`width="600"`, `cellpadding="0"`, `border="1"`, `colspan="2"`},
		},
		{
			name:     "img with dimensions",
			input:    `<img src="https://example.com/logo.png" width="200" height="100" alt="Logo">`,
			contains: []string{`width="200"`, `height="100"`, `alt="Logo"`, `src="https://example.com/logo.png"`},
		},
		{
			name:     "font tag preserved",
			input:    `<font color="red" size="3" face="Arial">Text</font>`,
			contains: []string{`color="red"`, `size="3"`, `face="Arial"`},
		},
		{
			name:     "html comment removed",
			input:    `<p>Before</p><!-- secret comment --><p>After</p>`,
			contains: []string{"<p>Before</p>", "<p>After</p>"},
			excludes: []string{"<!--", "secret"},
		},
		{
			name:     "nested script in allowed element",
			input:    `<div><script>alert(1)</script>Safe content</div>`,
			contains: []string{"<div>", "Safe content", "</div>"},
			excludes: []string{"<script", "alert"},
		},
		{
			name:     "case insensitive tag handling",
			input:    `<SCRIPT>alert(1)</SCRIPT><P>Text</P>`,
			contains: []string{"<p>", "Text", "</p>"},
			excludes: []string{"<script", "SCRIPT", "alert"},
		},
		{
			name:     "entity-encoded javascript URL",
			input:    `<a href="&#106;avascript:alert(1)">Click</a>`,
			contains: []string{"<a>", "Click"},
			excludes: []string{"javascript", "&#106;"},
		},
		{
			name:     "self-closing br and hr",
			input:    `Line 1<br/>Line 2<hr/>`,
			contains: []string{"Line 1", "<br />", "Line 2", "<hr />"},
		},
		{
			name:     "svg stripped entirely",
			input:    `<svg><circle cx="50" cy="50" r="40"/></svg><p>After</p>`,
			contains: []string{"<p>After</p>"},
			excludes: []string{"<svg", "circle"},
		},
		{
			name:     "heading elements",
			input:    `<h1>Title</h1><h2>Subtitle</h2><h6>Small</h6>`,
			contains: []string{"<h1>Title</h1>", "<h2>Subtitle</h2>", "<h6>Small</h6>"},
		},
		{
			name:     "list elements",
			input:    `<ul><li>Item 1</li><li>Item 2</li></ul>`,
			contains: []string{"<ul>", "<li>Item 1</li>", "<li>Item 2</li>", "</ul>"},
		},
		{
			name:     "relative URL allowed",
			input:    `<a href="/page">Link</a>`,
			contains: []string{`href="/page"`},
		},
		{
			name:     "fragment URL allowed",
			input:    `<a href="#section">Jump</a>`,
			contains: []string{`href="#section"`},
		},
		{
			name:     "global attributes preserved",
			input:    `<div class="container" id="main" dir="ltr" lang="en">Content</div>`,
			contains: []string{`class="container"`, `id="main"`, `dir="ltr"`, `lang="en"`},
		},
		{
			name:     "bgcolor global attribute",
			input:    `<td bgcolor="#ffffff">Cell</td>`,
			contains: []string{`bgcolor="#ffffff"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeHTML(tt.input)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got %q", want, result)
				}
			}
			for _, reject := range tt.excludes {
				if strings.Contains(result, reject) {
					t.Errorf("expected result to NOT contain %q, got %q", reject, result)
				}
			}
		})
	}
}

func TestSanitizeHTMLWithPolicy(t *testing.T) {
	t.Run("custom policy allows only b and i", func(t *testing.T) {
		p := NewPolicy().
			AllowElements("b", "i")

		result := SanitizeHTMLWithPolicy(`<b>Bold</b> <i>Italic</i> <u>Under</u> <script>alert(1)</script>`, p)

		if !strings.Contains(result, "<b>Bold</b>") {
			t.Errorf("expected <b> to be preserved, got %q", result)
		}
		if !strings.Contains(result, "<i>Italic</i>") {
			t.Errorf("expected <i> to be preserved, got %q", result)
		}
		if strings.Contains(result, "<u>") {
			t.Errorf("expected <u> to be stripped, got %q", result)
		}
		// Without StripElements, script tag is removed but content preserved.
		if strings.Contains(result, "<script") {
			t.Errorf("expected <script> tag removed, got %q", result)
		}
	})

	t.Run("custom policy with URL protocol restriction", func(t *testing.T) {
		p := NewPolicy().
			AllowAttributes("a", "href").
			AllowURLProtocols("href", "https")

		result := SanitizeHTMLWithPolicy(`<a href="https://safe.com">OK</a> <a href="http://unsafe.com">No</a>`, p)

		if !strings.Contains(result, `href="https://safe.com"`) {
			t.Errorf("expected https link preserved, got %q", result)
		}
		if strings.Contains(result, "http://unsafe.com") {
			t.Errorf("expected http link removed, got %q", result)
		}
	})

	t.Run("strip elements removes content", func(t *testing.T) {
		p := NewPolicy().
			AllowElements("p").
			StripElements("bad")

		result := SanitizeHTMLWithPolicy(`<p>OK</p><bad>Hidden content</bad><p>After</p>`, p)

		if !strings.Contains(result, "<p>OK</p>") {
			t.Errorf("expected <p>OK</p>, got %q", result)
		}
		if strings.Contains(result, "Hidden") {
			t.Errorf("expected stripped content removed, got %q", result)
		}
		if !strings.Contains(result, "<p>After</p>") {
			t.Errorf("expected <p>After</p>, got %q", result)
		}
	})
}

func TestPolicy_Builder(t *testing.T) {
	t.Run("AllowGlobalAttributes", func(t *testing.T) {
		p := NewPolicy().
			AllowElements("p", "span").
			AllowGlobalAttributes("class", "id")

		result := SanitizeHTMLWithPolicy(`<p class="x" id="y" onclick="bad()">Text</p>`, p)

		if !strings.Contains(result, `class="x"`) {
			t.Errorf("expected class preserved, got %q", result)
		}
		if !strings.Contains(result, `id="y"`) {
			t.Errorf("expected id preserved, got %q", result)
		}
		if strings.Contains(result, "onclick") {
			t.Errorf("expected onclick removed, got %q", result)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		p := NewPolicy().
			AllowElements("div").
			AllowAttributes("div", "class").
			AllowGlobalAttributes("id").
			StripElements("script").
			AllowAttributes("a", "href").
			AllowURLProtocols("href", "https")

		// Should not panic and produce a valid policy.
		result := SanitizeHTMLWithPolicy(`<div class="c" id="i">OK</div><script>bad</script><a href="https://x.com">Link</a>`, p)

		if !strings.Contains(result, `<div class="c" id="i">OK</div>`) {
			t.Errorf("expected div with attrs, got %q", result)
		}
		if strings.Contains(result, "bad") {
			t.Errorf("expected script content stripped, got %q", result)
		}
		if !strings.Contains(result, `href="https://x.com"`) {
			t.Errorf("expected link preserved, got %q", result)
		}
	})
}

func TestSanitizeCSS(t *testing.T) {
	tests := []struct {
		name  string
		input string
		empty bool
	}{
		{"safe css", "color: red; font-size: 14px", false},
		{"expression", "width: expression(alert(1))", true},
		{"javascript url", "background: url(javascript:alert(1))", true},
		{"vbscript", "background: vbscript:code", true},
		{"moz-binding", "-moz-binding: url(evil)", true},
		{"behavior", "behavior: url(evil.htc)", true},
		{"url http preserved", "background: url(http://example.com/img.png)", false},
		{"url https preserved", "background: url(https://example.com/img.png)", false},
		{"url cid preserved", "background: url(cid:logo@example.com)", false},
		{"url quoted preserved", `background: url("https://example.com/a.png")`, false},
		{"url relative preserved", "background: url(/img.png)", false},
		{"url data rejected", "background: url(data:image/png;base64,AAA)", true},
		{"url unterminated rejected", "background: url(", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeCSS(tt.input)
			if tt.empty && result != "" {
				t.Errorf("expected empty result for dangerous CSS, got %q", result)
			}
			if !tt.empty && result == "" {
				t.Errorf("expected CSS to be preserved, got empty string")
			}
		})
	}
}

func TestSanitizeFuncMap(t *testing.T) {
	funcMap := SanitizeFuncMap()

	tmpl, err := htmltemplate.New("test").Funcs(funcMap).Parse(`{{.Content | sanitize}}`)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	var buf strings.Builder
	data := map[string]any{
		"Content": `<b>Bold</b><script>alert(1)</script>`,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<b>Bold</b>") {
		t.Errorf("expected <b> preserved, got %q", result)
	}
	if strings.Contains(result, "<script") || strings.Contains(result, "alert") {
		t.Errorf("expected script removed, got %q", result)
	}
}

func TestSanitizeFuncMapWithPolicy(t *testing.T) {
	p := NewPolicy().AllowElements("i")
	funcMap := SanitizeFuncMapWithPolicy(p)

	tmpl, err := htmltemplate.New("test").Funcs(funcMap).Parse(`{{.Content | sanitize}}`)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	var buf strings.Builder
	data := map[string]any{
		"Content": `<i>Italic</i><b>Bold</b>`,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<i>Italic</i>") {
		t.Errorf("expected <i> preserved, got %q", result)
	}
	if strings.Contains(result, "<b>") {
		t.Errorf("expected <b> removed, got %q", result)
	}
}

func TestWithSanitizationMiddleware(t *testing.T) {
	mock := NewMockSender()
	wrapped := Chain(mock, WithSanitization())

	e := NewEmail().
		SetFrom("test@example.com").
		AddTo("user@example.com").
		SetSubject("Test").
		SetHTMLBody(`<p>Safe</p><script>alert(1)</script>`)

	builtEmail, err := e.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if err := wrapped.Send(context.Background(), builtEmail); err != nil {
		t.Fatalf("send: %v", err)
	}

	sent := mock.GetLastEmail()
	if !strings.Contains(sent.HTMLBody, "<p>Safe</p>") {
		t.Errorf("expected <p>Safe</p> preserved, got %q", sent.HTMLBody)
	}
	if strings.Contains(sent.HTMLBody, "<script") || strings.Contains(sent.HTMLBody, "alert") {
		t.Errorf("expected script removed, got %q", sent.HTMLBody)
	}
}

func TestWithSanitizationMiddleware_NoHTMLBody(t *testing.T) {
	mock := NewMockSender()
	wrapped := Chain(mock, WithSanitization())

	e := NewEmail().
		SetFrom("test@example.com").
		AddTo("user@example.com").
		SetSubject("Test").
		SetBody("Plain text only")

	builtEmail, err := e.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if err := wrapped.Send(context.Background(), builtEmail); err != nil {
		t.Fatalf("send: %v", err)
	}

	sent := mock.GetLastEmail()
	if sent.Body != "Plain text only" {
		t.Errorf("expected plain text preserved, got %q", sent.Body)
	}
}

func TestWithSanitizationPolicyMiddleware(t *testing.T) {
	p := NewPolicy().AllowElements("b").StripElements("script")
	mock := NewMockSender()
	wrapped := Chain(mock, WithSanitizationPolicy(p))

	e := NewEmail().
		SetFrom("test@example.com").
		AddTo("user@example.com").
		SetSubject("Test").
		SetHTMLBody(`<b>Bold</b><p>Para</p><script>bad</script>`)

	builtEmail, err := e.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if err := wrapped.Send(context.Background(), builtEmail); err != nil {
		t.Fatalf("send: %v", err)
	}

	sent := mock.GetLastEmail()
	if !strings.Contains(sent.HTMLBody, "<b>Bold</b>") {
		t.Errorf("expected <b> preserved, got %q", sent.HTMLBody)
	}
	if strings.Contains(sent.HTMLBody, "<p>") {
		t.Errorf("expected <p> removed by custom policy, got %q", sent.HTMLBody)
	}
	if strings.Contains(sent.HTMLBody, "bad") {
		t.Errorf("expected script content stripped, got %q", sent.HTMLBody)
	}
}

func TestTemplateSanitization(t *testing.T) {
	t.Run("WithSanitization on template", func(t *testing.T) {
		tmpl := NewTemplate("test")
		tmpl.SetHTMLTemplate(`<div>{{.Content}}</div>`)
		tmpl.WithSanitization()

		email, err := tmpl.Render(map[string]any{
			"Content": "Hello",
		})
		if err != nil {
			t.Fatalf("render: %v", err)
		}

		if !strings.Contains(email.HTMLBody, "<div>") {
			t.Errorf("expected <div> preserved, got %q", email.HTMLBody)
		}
	})

	t.Run("WithSanitizationPolicy on template", func(t *testing.T) {
		p := NewPolicy().AllowElements("span")
		tmpl := NewTemplate("test")
		tmpl.SetHTMLTemplate(`<span>OK</span><div>Removed</div>`)
		tmpl.WithSanitizationPolicy(p)

		email, err := tmpl.Render(nil)
		if err != nil {
			t.Fatalf("render: %v", err)
		}

		if !strings.Contains(email.HTMLBody, "<span>OK</span>") {
			t.Errorf("expected <span> preserved, got %q", email.HTMLBody)
		}
		if strings.Contains(email.HTMLBody, "<div>") {
			t.Errorf("expected <div> removed, got %q", email.HTMLBody)
		}
	})

	t.Run("no sanitization by default", func(t *testing.T) {
		tmpl := NewTemplate("test")
		tmpl.SetHTMLTemplate(`<blink>Text</blink>`)

		email, err := tmpl.Render(nil)
		if err != nil {
			t.Fatalf("render: %v", err)
		}

		// Without sanitization, raw output is preserved.
		if !strings.Contains(email.HTMLBody, "<blink>") {
			t.Errorf("expected <blink> preserved without sanitization, got %q", email.HTMLBody)
		}
	})
}

func TestSanitizeHTML_EdgeCases(t *testing.T) {
	t.Run("malformed unclosed tag", func(t *testing.T) {
		result := SanitizeHTML(`<p>Text<br`)
		// Should not panic; malformed part is escaped.
		if !strings.Contains(result, "Text") {
			t.Errorf("expected Text in output, got %q", result)
		}
	})

	t.Run("unclosed comment", func(t *testing.T) {
		result := SanitizeHTML(`<p>Before</p><!-- unclosed comment`)
		if !strings.Contains(result, "<p>Before</p>") {
			t.Errorf("expected content before comment, got %q", result)
		}
	})

	t.Run("only text", func(t *testing.T) {
		result := SanitizeHTML("Just plain text, no tags.")
		if result != "Just plain text, no tags." {
			t.Errorf("expected unchanged text, got %q", result)
		}
	})

	t.Run("empty tags", func(t *testing.T) {
		result := SanitizeHTML("<p></p>")
		if result != "<p></p>" {
			t.Errorf("expected <p></p>, got %q", result)
		}
	})

	t.Run("attribute with special chars in value", func(t *testing.T) {
		result := SanitizeHTML(`<a href="https://example.com?a=1&amp;b=2">Link</a>`)
		if !strings.Contains(result, "href=") {
			t.Errorf("expected href preserved, got %q", result)
		}
	})

	t.Run("nested disallowed tags", func(t *testing.T) {
		result := SanitizeHTML(`<article><section><p>Text</p></section></article>`)
		if !strings.Contains(result, "<p>Text</p>") {
			t.Errorf("expected <p>Text</p> preserved, got %q", result)
		}
		if strings.Contains(result, "<article") || strings.Contains(result, "<section") {
			t.Errorf("expected article/section removed, got %q", result)
		}
	})
}

func TestEmailPolicy(t *testing.T) {
	p := EmailPolicy()

	// Verify key allowed elements.
	allowedElements := []string{
		"p", "div", "span", "br", "hr", "table", "tr", "td", "th",
		"h1", "h2", "h3", "strong", "b", "em", "i", "u",
		"ul", "ol", "li", "a", "img", "font", "blockquote", "pre", "code",
	}
	for _, el := range allowedElements {
		if !p.isElementAllowed(el) {
			t.Errorf("expected %q to be allowed", el)
		}
	}

	// Verify strip elements.
	stripElements := []string{
		"script", "style", "iframe", "object", "embed",
		"form", "input", "textarea", "select", "button",
		"svg", "math", "applet",
	}
	for _, el := range stripElements {
		if !p.isStripElement(el) {
			t.Errorf("expected %q to be strip element", el)
		}
	}

	// Verify href protocols.
	if !p.isURLSafe("href", "https://example.com") {
		t.Error("expected https href to be safe")
	}
	if !p.isURLSafe("href", "mailto:user@example.com") {
		t.Error("expected mailto href to be safe")
	}
	if p.isURLSafe("href", "javascript:alert(1)") {
		t.Error("expected javascript href to be unsafe")
	}

	// Verify src protocols.
	if !p.isURLSafe("src", "https://example.com/img.png") {
		t.Error("expected https src to be safe")
	}
	if !p.isURLSafe("src", "cid:image001") {
		t.Error("expected cid src to be safe")
	}
	if p.isURLSafe("src", "data:image/png;base64,...") {
		t.Error("expected data src to be unsafe")
	}
}

func BenchmarkSanitizeHTML(b *testing.B) {
	input := `
		<div style="max-width: 600px; margin: 0 auto;">
			<h1>Welcome!</h1>
			<p>Hello <strong>World</strong></p>
			<a href="https://example.com">Click here</a>
			<img src="https://example.com/logo.png" alt="Logo" width="200">
			<script>alert('xss')</script>
			<table width="100%"><tr><td>Cell</td></tr></table>
			<style>.evil { background: url(javascript:alert(1)); }</style>
		</div>
	`

	for b.Loop() {
		SanitizeHTML(input)
	}
}
