package mail

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"time"
)

// dialTimeout bounds how long SMTPMailer waits to establish the TCP
// connection and complete the whole SMTP exchange. net/smtp is a frozen
// standard-library package (see SMTPMailer's doc comment) and
// smtp.SendMail applies no timeout of its own, so SMTPMailer drives the
// connection manually with an explicit deadline instead of using it.
const dialTimeout = 10 * time.Second

// SMTPMailer sends plain-text mail over SMTP, unauthenticated — sufficient
// for the dev-only Mailpit target this ticket scopes to (#42). Production
// SMTP wiring (auth, TLS, a real provider) is an explicit deployment
// concern, out of scope here.
//
// net/smtp is frozen upstream and smtp.SendMail assumes no I/O timeout, so
// SMTPMailer dials with a net.Dialer, sets an explicit connection deadline,
// and drives smtp.NewClient over that connection itself rather than calling
// smtp.SendMail.
type SMTPMailer struct {
	addr    string
	from    string
	baseURL *url.URL
}

var _ Mailer = (*SMTPMailer)(nil)

// NewSMTPMailer builds an SMTPMailer targeting addr (host:port) with from
// as the envelope/header From address. baseURL must already be validated
// (internal/config.Load parses and validates APP_BASE_URL at startup).
func NewSMTPMailer(addr, from string, baseURL *url.URL) *SMTPMailer {
	return &SMTPMailer{addr: addr, from: from, baseURL: baseURL}
}

func (m *SMTPMailer) SendVerificationEmail(ctx context.Context, to, token string) error {
	link := buildLink(m.baseURL, verifyEmailPath, token)
	return m.send(ctx, to, "Verify your email", verificationEmailBody(link, token))
}

func (m *SMTPMailer) SendPasswordResetEmail(ctx context.Context, to, token string) error {
	link := buildLink(m.baseURL, resetPasswordPath, token)
	return m.send(ctx, to, "Reset your password", passwordResetEmailBody(link, token))
}

// send opens one connection, sends one message, and closes the connection.
// It never logs or returns the token or message body — callers (the
// async-send goroutine in internal/http) are responsible for logging only a
// non-sensitive summary on failure.
func (m *SMTPMailer) send(ctx context.Context, to, subject, body string) error {
	deadline := time.Now().Add(dialTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	dialer := &net.Dialer{Deadline: deadline}
	conn, err := dialer.DialContext(ctx, "tcp", m.addr)
	if err != nil {
		return fmt.Errorf("mail: dial %s: %w", m.addr, err)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return fmt.Errorf("mail: set deadline: %w", err)
	}

	host, _, splitErr := net.SplitHostPort(m.addr)
	if splitErr != nil {
		host = m.addr
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mail: new client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("mail: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mail: RCPT TO: %w", err)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: DATA: %w", err)
	}
	if _, err := wc.Write(buildMessage(m.from, to, subject, body)); err != nil {
		_ = wc.Close()
		return fmt.Errorf("mail: write message: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("mail: close data: %w", err)
	}

	return client.Quit()
}

// buildMessage renders a minimal RFC 5322 message: headers, a blank line,
// then the plain-text body. Lines are CRLF-terminated as net/smtp expects;
// dot-stuffing of body lines starting with "." is handled by the
// DotWriter returned from Client.Data(), not here.
func buildMessage(from, to, subject, body string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("\r\n")
	for _, line := range splitLines(body) {
		buf.WriteString(line)
		buf.WriteString("\r\n")
	}
	return buf.Bytes()
}

// splitLines splits body on "\n", trimming any trailing "\r" so mixed
// line-ending input doesn't produce doubled carriage returns once
// buildMessage re-appends "\r\n".
func splitLines(body string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(body); i++ {
		if body[i] == '\n' {
			line := body[start:i]
			line = trimTrailingCR(line)
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(body) {
		lines = append(lines, trimTrailingCR(body[start:]))
	}
	return lines
}

func trimTrailingCR(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}
