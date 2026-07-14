# 0007 — JWT signing is HS256 with a single static secret

- **Migration ticket:** #41 (https://github.com/GonzaloSecades/nuchi/issues/41)
- **Area:** auth
- **Priority guess:** medium

## How it was migrated

Owned auth (new construction replacing Clerk — no legacy port) signs access
tokens with HS256 and one symmetric secret from `AUTH_JWT_SECRET`. Claims
are minimal (`sub`, `iat`, `exp`); no `kid` header.

## Why it was done this way

One service signs and verifies, so asymmetric keys would add ceremony
without benefit today; agreed in the #41 design review ("half-open gate",
Note A of the extensibility table). Fail-fast startup enforces the secret's
presence and minimum length.

## The concern

Two coupled limitations: (1) rotating the secret instantly invalidates every
live access token — there is no `kid` header, so dual-secret grace-window
acceptance needs a small code change when rotation is first wanted; (2) any
component holding the secret can mint tokens, so the secret must never be
shared with a second service.

## Proposed improvement

When secret rotation becomes routine (or a second token consumer appears —
mobile BFF, webhook service): add a `kid` header + keyed secret set for
graceful rotation, or switch to asymmetric signing (Ed25519/RS256) with the
private key confined to the API. Zero data migration either way — tokens
live 30 minutes and nothing persists them.
