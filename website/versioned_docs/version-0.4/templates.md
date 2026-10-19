---
id: templates
title: Templates
description: Register and render HTML and plain-text email templates with Go's text/template and html/template.
---

# Templates

Templates use Go's `text/template` and `html/template` packages, so you
get the same syntax and HTML-context auto-escaping you're used to. A
single `Template` bundles a subject, an optional plain-text body, and an
optional HTML body. Register templates by name with
`Mailer.RegisterTemplate` and render them with `Mailer.SendTemplate`.

## Creating Templates

```go
tmpl := email.NewTemplate("welcome")
tmpl.SetSubject("Welcome {{.Name}}!")

tmpl.SetTextTemplate(`Hello {{.Name}},

Thanks for signing up. Confirm your address: {{.VerifyLink}}

— The Team
`)

tmpl.SetHTMLTemplate(`<!DOCTYPE html>
<html>
  <body>
    <h1>Hello {{.Name}}!</h1>
    <p>Thanks for signing up. Click below to verify:</p>
    <a href="{{.VerifyLink}}">Verify Email</a>
  </body>
</html>`)

mailer.RegisterTemplate("welcome", tmpl)
```

## Loading from Files

For non-trivial layouts, keep templates in their own files and load them
at startup.

```go
tmpl, err := email.LoadTemplateFromFile("welcome", "templates/welcome.html")
if err != nil {
    log.Fatal(err)
}

mailer.RegisterTemplate("welcome", tmpl)
```

## Rendering

Pass any value as data — a struct, a map, or a primitive. Use a struct for
compile-time safety and IDE completion in real applications:

```go
type WelcomeData struct {
    Name       string
    VerifyLink string
}

err := mailer.SendTemplate(ctx,
    []string{"alice@example.com"},
    "welcome",
    WelcomeData{
        Name:       "Alice",
        VerifyLink: "https://example.com/verify/abc123",
    },
)
```

**Note:** the HTML template is rendered through `html/template`, which
auto-escapes `{{.UserInput}}` as text in HTML, attribute, and URL
contexts. Don't sidestep this with `template.HTML` on untrusted input —
it's an XSS vector.
