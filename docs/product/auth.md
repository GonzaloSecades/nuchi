# Authentication — Product Behavior

What each auth screen does, what a user sees when something goes wrong, and the
three behaviors that look like bugs but are deliberate.

Written for product decisions and support answers. Endpoint shapes are in
[the contract](../../openapi/nuchi.openapi.json); the pages themselves live in
`app/(auth)/`.

Replaces Clerk's hosted widgets as of
[#51](https://github.com/GonzaloSecades/nuchi/issues/51).

## The screens

| Route              | Purpose                                              |
| ------------------ | ---------------------------------------------------- |
| `/sign-in`         | Email and password. The only way into the dashboard. |
| `/sign-up`         | Creates the account and sends a verification email.  |
| `/verify-email`    | Where the emailed verification link lands.           |
| `/forgot-password` | Requests a reset email.                              |
| `/reset-password`  | Where the emailed reset link lands.                  |

`/verify-email` and `/reset-password` are **not free choices** — the Go mailer
builds those URLs (`backend/internal/mail/mail.go`), so renaming either route
breaks every link already sitting in a user's inbox.

## Signing up does not sign you in

Registration creates the account and stops there. The new user is sent to a
"check your email" screen, not the dashboard.

Attempting to sign in before verifying is refused with a specific message
pointing back at the inbox — the API answers `403 EMAIL_NOT_VERIFIED`, which
the sign-in page phrases as _"Verify your email before signing in."_ This is the
most likely "I can't log in" support ticket, and it is working as intended.

There is no "resend verification email" button. Signing up again with the same
address issues a fresh link.

## The reset form always says the same thing

`/forgot-password` shows the identical confirmation whether or not the address
belongs to a real account: _"If an account exists for that email, a password
reset link has been sent."_

This is deliberate and worth protecting. If the screen distinguished the two
cases, anyone could use the form to discover which email addresses have Nuchi
accounts, one guess at a time. A support answer along the lines of "it said the
email was sent, so the account exists" is therefore **wrong** — the message says
nothing either way.

The same reasoning applies to sign-in: a wrong password and an unknown email
both return _"Invalid email or password."_

Reset emails are capped at **three per hour per account**. Past the cap the
screen still shows the same confirmation and no email arrives.

## Resetting a password signs you out everywhere

A completed reset revokes every outstanding session on the account, on every
device. The user is then sent to `/sign-in` to start a new one.

This is what makes a reset useful after a suspected compromise: it ends the
attacker's session rather than leaving it running alongside the new password.

## Staying signed in

A session survives closing the tab and reopening it, but it is re-established on
each page load rather than simply being remembered — which is why the dashboard
shows a brief spinner before it appears. That moment is the app asking the
server whether the session is still good. The sign-in screen waits for the same
answer; if the session is valid, the app returns to the requested dashboard
page instead of showing the sign-in form.

Sessions end when the user signs out, when a password reset revokes them, or
when the refresh token expires. When one ends mid-session the user is returned
to `/sign-in`, and work in progress on screen is lost — there is no draft
recovery.

## Signing out

The account menu in the dashboard header carries the only sign-out control. It
revokes the session server-side, not just locally, so a stolen refresh token
stops working at that moment.

If the server cannot be reached, the user is still signed out locally and told
so. The local session is gone either way; the server-side token would then
remain valid until it expires on its own.

## What the pages deliberately don't do

- **No social or SSO sign-in.** Email and password only.
- **No profile screen.** The API stores an id, an email and a verified flag —
  there is no display name, which is also why the dashboard greeting says
  "Welcome back" without one.
- **No password change while signed in.** Changing a password goes through the
  reset flow.
- **No account deletion.**

Each of these was in Clerk's widget and is not in the API. They are product
gaps to schedule, not regressions to file.

## Local testing

Verification and reset emails are real SMTP sends. In development they are
caught by Mailpit rather than delivered — open <http://localhost:8025> to read
them and follow the links.
