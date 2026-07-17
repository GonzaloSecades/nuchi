# 0012 — Residual timing oracle on password-reset request

- **Migration ticket:** #42 (https://github.com/GonzaloSecades/nuchi/issues/42)
- **Area:** auth
- **Priority guess:** low

## How it was migrated

`POST /auth/password-reset/request` returns the same 200 body whether or not
the email belongs to a real account, and email delivery is moved off the
response path (async), so response time no longer varies with SMTP latency.
But an existing account still runs a small serialized DB transaction (lock,
cap check, invalidate prior tokens, create token) that a non-existent
account skips entirely.

## Why it was done this way

Enumeration safety via identical responses + async delivery is the
OWASP-recommended baseline and was agreed in the #42 review. Making the
unknown-account path do *identical* work (a fake locked transaction against
a decoy row) is more complexity and its own risks; the review explicitly
accepted the residual difference as debt rather than claim full timing
safety.

## The concern

A determined attacker measuring response-time distributions could still
distinguish real from unknown emails by the presence/absence of the DB
transaction — a narrower oracle than an SMTP-blocking send, but non-zero.
For a personal-scale app the practical risk is low; recorded so it is a
decision, not an oversight.

## Proposed improvement

Options, post-parity: (a) perform equivalent decoy work for unknown emails
(constant-time-ish path), or (b) move all reset issuance fully async behind
a queue so the handler does uniform trivial work in every case. OWASP
Forgot-Password guidance: uniform response time or async delivery.
Reference: https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html
