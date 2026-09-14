---
id: webhooks
title: Webhooks
description: Receive and normalize delivery events (bounces, opens, clicks) from any provider.
---

# Webhooks

Once an email leaves your process, the only way to know what happened —
delivered, bounced, opened, marked as spam — is for the provider to call
back. goemail provides a small provider-agnostic webhook layer: a
normalized `WebhookEvent` type, a `WebhookReceiver` HTTP handler, and
parser plugins per provider.

## Event Types

Provider-specific payloads are normalized into a single `EventType` enum
so your application code does not branch on provider:

| EventType | Meaning |
| --- | --- |
| `EventDelivered` | Accepted by recipient mail server |
| `EventBounced` | Hard bounce (permanent failure) |
| `EventDeferred` | Soft bounce (temporary failure) |
| `EventOpened` | Recipient opened the email |
| `EventClicked` | Recipient clicked a tracked link |
| `EventComplained` | Recipient marked the email as spam |
| `EventUnsubscribed` | Recipient unsubscribed |
| `EventDropped` | Provider rejected the message before sending |

## WebhookReceiver

`WebhookReceiver` is an `http.Handler`. Mount it on the route your
provider posts to, and pass it a `WebhookHandler` that processes
normalized events. Returning an error causes the receiver to respond
`500` so the provider retries.

```go
handler := email.WebhookHandlerFunc(func(ctx context.Context, ev email.WebhookEvent) error {
    switch ev.Type {
    case email.EventBounced:
        return suppressionList.Add(ev.Recipient, ev.Reason)
    case email.EventComplained:
        return spamComplaints.Record(ev.Recipient)
    case email.EventOpened, email.EventClicked:
        analytics.Track(ev)
    }
    return nil
})

receiver := email.NewWebhookReceiver(parser, handler,
    email.WithWebhookLogger(logger),
    email.WithEventFilter(
        email.EventBounced,
        email.EventComplained,
        email.EventDelivered,
    ),
)

http.Handle("/webhooks/email", receiver)
```

## Provider Parsers

Each provider has its own webhook payload format and signature scheme.
Parser submodules validate the signature and convert the payload into
`[]WebhookEvent`. Pass the parser to `NewWebhookReceiver` as shown above.

Provider parser modules are released under `providers/webhook<provider>`
as they become available — track [the repository](https://github.com/KARTIKrocks/goemail)
for status.
