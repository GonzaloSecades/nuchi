# 0011 — No resend endpoint; email delivery is fire-and-forget

- **Migration ticket:** #42 (https://github.com/GonzaloSecades/nuchi/issues/42)
- **Area:** auth / infra
- **Priority guess:** medium

## How it was migrated

Verification and reset emails are sent asynchronously and best-effort:
after the token-creating transaction commits, a goroutine attempts the SMTP
send; a failure is logged (never the token) and never affects the response.
There is no resend endpoint (none exists in the OpenAPI contract), and no
durable delivery (no outbox, no retry).

## Why it was done this way

Async best-effort send is the deliberate mitigation for the reset-request
timing oracle (response returns before SMTP work), agreed in the #42 design
review. Register must still 201 and reset-request must still 200 regardless
of delivery, or an SMTP-dependent error would leak account existence.
Adding endpoints or an outbox is contract/scope growth beyond parity.

## The concern

A registered account whose verification email fails to send is stranded:
the user is unverified, cannot log in (login requires verification), and has
no way to request a new email. Fire-and-forget also means a transient SMTP
blip silently drops the only delivery attempt.

## Proposed improvement

Add `POST /auth/verify-email/resend` and `/auth/password-reset/request` is
already the reset resend path. Back delivery with a durable outbox table
(the token tables' `created_at` and a small `email_outbox` row) plus a
retry worker, so sends survive a transient SMTP outage. Contract change +
new table; post-parity. Until then, monitoring on the send-failure log line
is the operational stopgap.
