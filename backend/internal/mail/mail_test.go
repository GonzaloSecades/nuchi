package mail

import (
	"context"
	"io"
	"net"
	"net/textproto"
	"net/url"
	"strings"
	"testing"
	"time"
)

// startFakeSMTPServer accepts exactly one connection, speaks just enough
// SMTP to satisfy net/smtp.Client (EHLO/HELO, MAIL, RCPT, DATA, QUIT), and
// publishes the un-dot-stuffed DATA body to the returned channel. It has no
// dependency on a real Mailpit instance — SMTPMailer's tests only need to
// prove the client speaks correct SMTP and builds the right message.
func startFakeSMTPServer(t *testing.T) (addr string, dataCh <-chan string) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	ch := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		tp := textproto.NewConn(conn)
		_ = tp.PrintfLine("220 fake.smtp ESMTP ready")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			upper := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
				_ = tp.PrintfLine("250 fake.smtp")
			case strings.HasPrefix(upper, "MAIL FROM"):
				_ = tp.PrintfLine("250 OK")
			case strings.HasPrefix(upper, "RCPT TO"):
				_ = tp.PrintfLine("250 OK")
			case upper == "DATA":
				_ = tp.PrintfLine("354 End data with <CR><LF>.<CR><LF>")
				body, _ := io.ReadAll(tp.DotReader())
				select {
				case ch <- string(body):
				default:
				}
				_ = tp.PrintfLine("250 OK: queued")
			case upper == "QUIT":
				_ = tp.PrintfLine("221 Bye")
				return
			default:
				_ = tp.PrintfLine("250 OK")
			}
		}
	}()

	return ln.Addr().String(), ch
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	return u
}

func TestSMTPMailer_SendVerificationEmail(t *testing.T) {
	addr, dataCh := startFakeSMTPServer(t)
	m := NewSMTPMailer(addr, "nuchi@localhost", mustParseURL(t, "http://localhost:3000"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.SendVerificationEmail(ctx, "user@example.test", "tok-abc123"); err != nil {
		t.Fatalf("SendVerificationEmail: unexpected error: %v", err)
	}

	select {
	case body := <-dataCh:
		if !strings.Contains(body, "To: user@example.test") {
			t.Errorf("expected To header for recipient, got body:\n%s", body)
		}
		if !strings.Contains(body, "http://localhost:3000/verify-email?token=tok-abc123") {
			t.Errorf("expected verification link in body, got:\n%s", body)
		}
		if !strings.Contains(body, "tok-abc123") {
			t.Errorf("expected raw token on its own line for curl testing, got:\n%s", body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for captured SMTP DATA")
	}
}

func TestSMTPMailer_SendPasswordResetEmail(t *testing.T) {
	addr, dataCh := startFakeSMTPServer(t)
	m := NewSMTPMailer(addr, "nuchi@localhost", mustParseURL(t, "http://localhost:3000"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.SendPasswordResetEmail(ctx, "reset-user@example.test", "reset-tok-456"); err != nil {
		t.Fatalf("SendPasswordResetEmail: unexpected error: %v", err)
	}

	select {
	case body := <-dataCh:
		if !strings.Contains(body, "To: reset-user@example.test") {
			t.Errorf("expected To header for recipient, got body:\n%s", body)
		}
		if !strings.Contains(body, "http://localhost:3000/reset-password?token=reset-tok-456") {
			t.Errorf("expected reset link in body, got:\n%s", body)
		}
		if !strings.Contains(body, "reset-tok-456") {
			t.Errorf("expected raw token on its own line for curl testing, got:\n%s", body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for captured SMTP DATA")
	}
}

// TestSMTPMailer_ConnectionDeadlineEnforced proves SMTPMailer enforces its
// own deadline rather than hanging forever: net/smtp.SendMail is documented
// to apply no timeout, so SMTPMailer dials with net.Dialer and sets an
// explicit conn.SetDeadline instead. The fake server here accepts the
// connection but never speaks, which would hang indefinitely without that
// deadline.
func TestSMTPMailer_ConnectionDeadlineEnforced(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(3 * time.Second)
	}()

	m := NewSMTPMailer(ln.Addr().String(), "nuchi@localhost", mustParseURL(t, "http://localhost:3000"))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = m.SendVerificationEmail(ctx, "user@example.test", "tok")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error when the SMTP server never responds")
	}
	if elapsed > 2*time.Second {
		t.Errorf("expected the send to fail quickly via the enforced deadline, took %v", elapsed)
	}
}

func TestBuildLink_EscapesTokenQueryParam(t *testing.T) {
	link := buildLink(mustParseURL(t, "http://localhost:3000"), verifyEmailPath, "tok with space & stuff")

	want := "http://localhost:3000/verify-email?token=tok+with+space+%26+stuff"
	if link != want {
		t.Errorf("expected escaped link %q, got %q", want, link)
	}
}

func TestBuildLink_PreservesBaseURLHostAndScheme(t *testing.T) {
	link := buildLink(mustParseURL(t, "https://app.example.test"), resetPasswordPath, "abc123")

	want := "https://app.example.test/reset-password?token=abc123"
	if link != want {
		t.Errorf("expected %q, got %q", want, link)
	}
}

func TestCapturingMailer_RecordsSendsAndIsConcurrencySafe(t *testing.T) {
	m := NewCapturingMailer()

	const n = 20
	done := make(chan struct{}, n*2)
	for range n {
		go func() {
			_ = m.SendVerificationEmail(context.Background(), "v@example.test", "tok")
			done <- struct{}{}
		}()
		go func() {
			_ = m.SendPasswordResetEmail(context.Background(), "r@example.test", "tok")
			done <- struct{}{}
		}()
	}
	for range n * 2 {
		<-done
	}

	if got := len(m.VerificationSends()); got != n {
		t.Errorf("expected %d captured verification sends, got %d", n, got)
	}
	if got := len(m.ResetSends()); got != n {
		t.Errorf("expected %d captured reset sends, got %d", n, got)
	}
}

func TestCapturingMailer_ErrReturnsWithoutRecording(t *testing.T) {
	m := NewCapturingMailer()
	m.Err = context.DeadlineExceeded

	if err := m.SendVerificationEmail(context.Background(), "v@example.test", "tok"); err == nil {
		t.Fatal("expected the configured Err to be returned")
	}
	if got := len(m.VerificationSends()); got != 0 {
		t.Errorf("expected no captured send when Err is set, got %d", got)
	}
}
