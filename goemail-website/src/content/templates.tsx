import CodeBlock from '../components/CodeBlock';

export default function TemplatesDocs() {
  return (
    <section id="templates" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Templates</h2>
      <p className="text-text-muted mb-4">
        Templates use Go's <code>text/template</code> and <code>html/template</code> packages, so you
        get the same syntax and HTML-context auto-escaping you're used to. A single
        <code> Template</code> bundles a subject, an optional plain-text body, and an optional HTML
        body. Register templates by name with <code>Mailer.RegisterTemplate</code> and render them with
        <code> Mailer.SendTemplate</code>.
      </p>

      <h3 id="templates-creating" className="text-lg font-semibold text-text-heading mt-8 mb-2">Creating Templates</h3>
      <CodeBlock code={`tmpl := email.NewTemplate("welcome")
tmpl.SetSubject("Welcome {{.Name}}!")

tmpl.SetTextTemplate(\`Hello {{.Name}},

Thanks for signing up. Confirm your address: {{.VerifyLink}}

— The Team
\`)

tmpl.SetHTMLTemplate(\`<!DOCTYPE html>
<html>
  <body>
    <h1>Hello {{.Name}}!</h1>
    <p>Thanks for signing up. Click below to verify:</p>
    <a href="{{.VerifyLink}}">Verify Email</a>
  </body>
</html>\`)

mailer.RegisterTemplate("welcome", tmpl)`} />

      <h3 id="templates-files" className="text-lg font-semibold text-text-heading mt-8 mb-2">Loading from Files</h3>
      <p className="text-text-muted mb-3">
        For non-trivial layouts, keep templates in their own files and load them at startup.
      </p>
      <CodeBlock code={`tmpl, err := email.LoadTemplateFromFile("welcome", "templates/welcome.html")
if err != nil {
    log.Fatal(err)
}

mailer.RegisterTemplate("welcome", tmpl)`} />

      <h3 id="templates-rendering" className="text-lg font-semibold text-text-heading mt-8 mb-2">Rendering</h3>
      <p className="text-text-muted mb-3">
        Pass any value as data — a struct, a map, or a primitive. Use a struct for compile-time safety
        and IDE completion in real applications:
      </p>
      <CodeBlock code={`type WelcomeData struct {
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
)`} />

      <p className="text-text-muted mb-3 text-sm">
        <strong>Note:</strong> the HTML template is rendered through <code>html/template</code>, which
        auto-escapes <code>{`{{.UserInput}}`}</code> as text in HTML, attribute, and URL contexts.
        Don't sidestep this with <code>template.HTML</code> on untrusted input — it's an XSS vector.
      </p>
    </section>
  );
}
