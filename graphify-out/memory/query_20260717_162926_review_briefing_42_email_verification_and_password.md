---
type: "query"
date: "2026-07-17T16:29:26.967155+00:00"
question: "Review briefing 42 email verification and password reset design and confirm best-effort send, silent rate cap, TTL defaults, and plain-text future-frontend links"
contributor: "graphify"
outcome: "useful"
source_nodes: ["ConsumeEmailVerificationToken", "ConsumePasswordResetToken", "CountRecentPasswordResetTokensParams", "RevokeAllUserRefreshTokens", "Mailpit Local SMTP", "Config"]
---

# Q: Review briefing 42 email verification and password reset design and confirm best-effort send, silent rate cap, TTL defaults, and plain-text future-frontend links

## Answer

Expanded from original query via graph vocab: [email, verification, password, reset, token, consume, session, revocation, rate, limit, smtp, config]. Overall design is sound but requires amendments before implementation: token consume and the corresponding state mutation must occur in the same database transaction; reset issuance must serialize per user and make count, prior-token invalidation, and new-token creation atomic so concurrent requests cannot exceed the cap or leave multiple valid tokens. Best-effort responses, silent cap behavior, 48h/30m TTLs, and plain-text links are acceptable. Synchronous SMTP only for known accounts remains a timing enumeration channel, so use asynchronous delivery or explicitly record the limitation. Send only after commit. Add concurrent and rollback/fault tests.

## Outcome

- Signal: useful

## Source Nodes

- ConsumeEmailVerificationToken
- ConsumePasswordResetToken
- CountRecentPasswordResetTokensParams
- RevokeAllUserRefreshTokens
- Mailpit Local SMTP
- Config