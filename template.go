package email

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"
)

// Template represents an email template
type Template struct {
	name           string
	subject        string
	subjectTmpl    *texttemplate.Template
	textTmpl       *texttemplate.Template
	htmlTmpl       *htmltemplate.Template
	subjectErr     error // parse error from SetSubject, surfaced in Render
	sanitizePolicy *Policy
}

// NewTemplate creates a new email template
func NewTemplate(name string) *Template {
	return &Template{
		name: name,
	}
}

// SetSubject sets the subject template
func (t *Template) SetSubject(subject string) *Template {
	t.subject = subject
	t.subjectTmpl = nil
	t.subjectErr = nil
	if subject != "" {
		parsed, err := texttemplate.New(t.name + "_subject").Parse(subject)
		if err != nil {
			t.subjectErr = err
		} else {
			t.subjectTmpl = parsed
		}
	}
	return t
}

// SetTextTemplate sets the plain text template
func (t *Template) SetTextTemplate(tmpl string) (*Template, error) {
	parsed, err := texttemplate.New(t.name + "_text").Parse(tmpl)
	if err != nil {
		return nil, err
	}
	t.textTmpl = parsed
	return t, nil
}

// SetHTMLTemplate sets the HTML template
func (t *Template) SetHTMLTemplate(tmpl string) (*Template, error) {
	parsed, err := htmltemplate.New(t.name + "_html").Parse(tmpl)
	if err != nil {
		return nil, err
	}
	t.htmlTmpl = parsed
	return t, nil
}

// Render renders the template with data
func (t *Template) Render(data any) (*Email, error) {
	email := NewEmail()

	// Render subject
	if t.subject != "" {
		if t.subjectErr != nil {
			return nil, t.subjectErr
		}

		var subjBuf bytes.Buffer
		if err := t.subjectTmpl.Execute(&subjBuf, data); err != nil {
			return nil, err
		}
		email.Subject = subjBuf.String()
	}

	// Render text body
	if t.textTmpl != nil {
		var textBuf bytes.Buffer
		if err := t.textTmpl.Execute(&textBuf, data); err != nil {
			return nil, err
		}
		email.Body = textBuf.String()
	}

	// Render HTML body
	if t.htmlTmpl != nil {
		var htmlBuf bytes.Buffer
		if err := t.htmlTmpl.Execute(&htmlBuf, data); err != nil {
			return nil, err
		}
		email.HTMLBody = htmlBuf.String()
	}

	// Sanitize HTML body if a policy is set.
	if t.sanitizePolicy != nil && email.HTMLBody != "" {
		email.HTMLBody = SanitizeHTMLWithPolicy(email.HTMLBody, t.sanitizePolicy)
	}

	return email, nil
}

// WithSanitization enables HTML sanitization on rendered output using the
// default [EmailPolicy]. The HTMLBody is sanitized after template rendering.
func (t *Template) WithSanitization() *Template {
	t.sanitizePolicy = defaultPolicy
	return t
}

// WithSanitizationPolicy enables HTML sanitization on rendered output using
// a custom [Policy].
func (t *Template) WithSanitizationPolicy(p *Policy) *Template {
	t.sanitizePolicy = p
	return t
}

// LoadTemplateFromFile loads a template from a file.
// The file content is used as the HTML template.
func LoadTemplateFromFile(name, path string) (*Template, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tmpl := NewTemplate(name)
	if _, err := tmpl.SetHTMLTemplate(string(content)); err != nil {
		return nil, err
	}

	return tmpl, nil
}

// LoadTemplateFromFS loads a single template from an fs.FS.
// This works with embed.FS, os.DirFS, or any other fs.FS implementation.
// Files ending in .txt are loaded as the plain-text body; all others are
// loaded as the HTML body.
func LoadTemplateFromFS(fsys fs.FS, name, path string) (*Template, error) {
	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", path, err)
	}

	tmpl := NewTemplate(name)

	if strings.HasSuffix(path, ".txt") {
		if _, err := tmpl.SetTextTemplate(string(content)); err != nil {
			return nil, fmt.Errorf("parse text template %q: %w", path, err)
		}
	} else {
		if _, err := tmpl.SetHTMLTemplate(string(content)); err != nil {
			return nil, fmt.Errorf("parse html template %q: %w", path, err)
		}
	}

	return tmpl, nil
}

// LoadTemplatesFromDir loads all templates from a directory on the local
// filesystem that match the given glob patterns (e.g. "*.html", "*.txt",
// "*.subject"). Multiple patterns can be provided and their results are merged.
//
// Template names are derived from the filename without extension. Files that
// share the same base name but differ in extension are merged into a single
// Template: .html files become the HTML body, .txt files become the
// plain-text body, and .subject files become the subject line. For example,
// given "welcome.html", "welcome.txt", and "welcome.subject", one Template
// named "welcome" is returned with all three fields populated.
func LoadTemplatesFromDir(dir string, patterns ...string) (map[string]*Template, error) {
	return LoadTemplatesFromFS(os.DirFS(dir), patterns...)
}

// LoadTemplatesFromFS loads all templates from an fs.FS that match the given
// glob patterns (e.g. "*.html", "*.txt", "*.subject"). Multiple patterns can
// be provided and their results are merged.
//
// This works with embed.FS, os.DirFS, or any fs.FS implementation.
//
// Template names are derived from the filename without extension. Files that
// share the same base name but differ in extension are merged into a single
// Template: .html files become the HTML body, .txt files become the
// plain-text body, and .subject files become the subject line. For example,
// given "welcome.html", "welcome.txt", and "welcome.subject", one Template
// named "welcome" is returned with all three fields populated.
//
// .subject files are read literally — leading/trailing whitespace is trimmed
// so a trailing newline at end-of-file is not preserved in the subject.
func LoadTemplatesFromFS(fsys fs.FS, patterns ...string) (map[string]*Template, error) {
	var matches []string
	for _, pattern := range patterns {
		m, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", pattern, err)
		}
		matches = append(matches, m...)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no templates matched patterns %v", patterns)
	}

	templates := make(map[string]*Template)

	for _, path := range matches {
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, fmt.Errorf("read template %q: %w", path, err)
		}

		ext := filepath.Ext(path)
		name := strings.TrimSuffix(filepath.Base(path), ext)

		tmpl, ok := templates[name]
		if !ok {
			tmpl = NewTemplate(name)
			templates[name] = tmpl
		}

		switch ext {
		case ".txt":
			if _, err := tmpl.SetTextTemplate(string(content)); err != nil {
				return nil, fmt.Errorf("parse text template %q: %w", path, err)
			}
		case ".html", ".htm", ".gohtml", ".tmpl":
			if _, err := tmpl.SetHTMLTemplate(string(content)); err != nil {
				return nil, fmt.Errorf("parse html template %q: %w", path, err)
			}
		case ".subject":
			subject := strings.TrimSpace(string(content))
			tmpl.SetSubject(subject)
			if tmpl.subjectErr != nil {
				return nil, fmt.Errorf("parse subject template %q: %w", path, tmpl.subjectErr)
			}
		default:
			return nil, fmt.Errorf("unsupported template extension %q for file %q", ext, path)
		}
	}

	return templates, nil
}
