# Security Policy

goemail signs and sends outbound mail with credentials and key material you
provide, so header/CRLF injection, credential handling, and DKIM signing are
treated as part of the API rather than as configuration details.

## Reporting a vulnerability

Please report privately through
[GitHub Security Advisories](https://github.com/KARTIKrocks/goemail/security/advisories/new)
rather than opening a public issue, so a fix can ship before the details are
public.

Useful things to include, if you have them: the affected version, whether the
issue is reachable from untrusted input (e.g. a recipient address or header
value a user supplied), and a minimal `Email`/`SMTPConfig` that reproduces it.

You can expect an acknowledgement within 7 days. If a report is confirmed, the
advisory is published together with the release that fixes it, and you will be
credited unless you ask otherwise.

## Supported versions

goemail is pre-1.0. Per semver, anything may change between minors, so
security fixes ship on the latest `0.x` minor only — there is no older line to
backport to.

| Version | Supported |
| --- | --- |
| Latest `0.x` minor | Yes |
| Anything older | No — upgrade |

`v0.2.0` is the worked example: it enforced a minimum of TLS 1.2 on STARTTLS
connections and closed several CRLF/header-injection gaps that `v0.1.0` did
not catch. See [CHANGELOG.md](CHANGELOG.md) for the details before taking a
fix if you're pinned to an older version.

The provider sub-modules (`providers/sendgrid`, `providers/mailgun`,
`providers/ses`, `providers/otelmail`) are versioned independently, and the
same rule applies to each at its own latest minor.

## Scope

In scope — issues reachable from input this library did not choose to trust:

- Header or CRLF injection via `To`/`Cc`/`Bcc`/`Subject`/custom headers or
  attachment filenames/content types
- Address validation bypass that lets malformed or hostile input reach an
  outgoing message
- DKIM signing or verification producing an invalid or spoofable signature
- TLS downgrade, or STARTTLS not being enforced when `UseTLS` is set
- Panics or data races reachable from building/sending a single `Email`
- Credential or key material (SMTP password, DKIM private key) being logged,
  retained, or exposed beyond its intended use

Out of scope:

- Misconfiguration that disables a protection deliberately, such as setting
  `UseTLS: false` or skipping TLS certificate verification on purpose
- Findings in `examples/`, which is illustrative code and not part of the
  supported surface
- Vulnerabilities in third-party dependencies — please report those upstream.
  Dependabot tracks them here and `govulncheck` gates CI on the ones this code
  actually reaches

## How this is checked

| Tool | Covers |
| --- | --- |
| [CodeQL](.github/workflows/codeql.yml) | Bug patterns in this repository's own Go code, scanned per module, on every push and pull request plus a weekly re-scan |
| [`govulncheck`](.github/workflows/ci.yml) | Known advisories in dependencies, filtered to those reachable from this code's call graph, run per module |
| [Dependabot](.github/dependabot.yml) | Dependency updates for the root module and each `providers/*` sub-module, plus the docs site and the GitHub Actions themselves |

CodeQL and `govulncheck` each run separately against the root module and the
four `providers/*` sub-modules, because a scan started from the root stops at
the nested `go.mod` boundaries and would otherwise miss the providers
entirely.
