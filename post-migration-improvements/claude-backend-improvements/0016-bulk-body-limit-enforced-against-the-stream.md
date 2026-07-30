# 0016 — Bulk body limits are enforced against the stream, not just Content-Length

- **Migration ticket:** #47 (https://github.com/GonzaloSecades/nuchi/issues/47)
- **Area:** http
- **Priority guess:** low

## How it was migrated

`decodeBulkBody` (`backend/internal/http/resources.go`) enforces each bulk
endpoint's byte limit twice:

1. `r.ContentLength > limit` — a cheap rejection for a client that declares an
   oversized body, mirroring legacy's `isContentLengthTooLarge` guard.
2. `http.MaxBytesReader` at the same limit — enforcement against the bytes that
   actually arrive.

Limits are unchanged from legacy: 1,000,000 bytes for bulk-create, 100,000 for
bulk-delete (`lib/transaction-limits.ts`), both answered with
`413 REQUEST_BODY_TOO_LARGE`.

## Why it was done this way

Legacy checks the `Content-Length` header and nothing else. A request that does
not send that header — any chunked request — passes the guard regardless of
size, and the body streams into the JSON decoder unbounded. The limit is
therefore advisory: it constrains clients that declare their size honestly and
no one else.

The contract describes the 413 as "the request body exceeds a documented bulk
operation size limit". A body that exceeds the limit without announcing itself
still exceeds the limit, so enforcing against the stream is the reading that
matches the documented behavior rather than an addition to it.

These are also the only endpoints in the migration with a real body cap at all.
The single-resource bodies are deliberately uncapped (entry 0013) because
neither legacy nor the contract bounds them; here the contract does, which is
what makes enforcement appropriate rather than an invented limit.

## The concern

This is a **behavior-visible divergence** for one input class: an oversized body
sent without a `Content-Length`. Legacy accepts and processes it; this
implementation answers `413`.

Practically that input is either a streaming client or an attacker, and the
frontend is neither — it sends `fetch` with a serialized string, so
`Content-Length` is always present. But it is a real difference and is recorded
rather than buried.

A second, smaller divergence is unavoidable rather than chosen: legacy also
ignores a **non-numeric** `Content-Length` and processes the request. That
branch is unreachable in Go — `net/http` parses the header and rejects a
malformed value with its own `400` before any handler runs — so it is neither
implemented nor tested. A test for it could only pass by constructing a request
that bypasses the server's parser, which would assert nothing about real
behavior.

## Proposed improvement

Nothing to undo; the Go behavior is the stricter and more honest one. Two
follow-ups worth folding into the post-parity hardening pass:

- Make the byte limits machine-readable. They are already stated in both bulk
  operation descriptions, so they are discoverable by a human reading the
  contract — but prose cannot be consumed by routing or middleware, and the
  handler constants can drift from it silently. A documented vendor extension
  (`x-max-request-bytes`) read by the router, or a single generated constant, is
  what would make the contract and the code impossible to disagree.
- Apply the same stream-enforced discipline to any future bulk endpoint by
  default, rather than per-handler. If more arrive, `decodeBulkBody` should
  become middleware keyed by route so a new bulk endpoint cannot silently ship
  without a cap.
