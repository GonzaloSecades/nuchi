package mail

import (
	"context"
	"sync"
)

// CapturedEmail records one call to a CapturingMailer method.
type CapturedEmail struct {
	To    string
	Token string
}

// CapturingMailer is a Mailer that records every send instead of talking to
// SMTP. It exists so tests (unit and live-DB HTTP tests) can assert what
// would have been sent — recipient and raw token — without standing up an
// SMTP server, and is the reason AuthServer takes a Mailer dependency
// instead of constructing an SMTPMailer directly (#42).
//
// Safe for concurrent use: the concurrent-issuance test fires several
// simultaneous reset requests, each triggering its own async send
// goroutine.
type CapturingMailer struct {
	mu                sync.Mutex
	verificationSends []CapturedEmail
	resetSends        []CapturedEmail

	// Err, when non-nil, is returned by both Send methods instead of
	// recording anything — used to exercise a send-failure path without a
	// real SMTP failure.
	Err error
}

var _ Mailer = (*CapturingMailer)(nil)

// NewCapturingMailer returns a ready-to-use CapturingMailer with no
// captured sends.
func NewCapturingMailer() *CapturingMailer {
	return &CapturingMailer{}
}

func (m *CapturingMailer) SendVerificationEmail(_ context.Context, to, token string) error {
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.verificationSends = append(m.verificationSends, CapturedEmail{To: to, Token: token})
	return nil
}

func (m *CapturingMailer) SendPasswordResetEmail(_ context.Context, to, token string) error {
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetSends = append(m.resetSends, CapturedEmail{To: to, Token: token})
	return nil
}

// VerificationSends returns a snapshot of every captured
// SendVerificationEmail call, in call order.
func (m *CapturingMailer) VerificationSends() []CapturedEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]CapturedEmail, len(m.verificationSends))
	copy(out, m.verificationSends)
	return out
}

// ResetSends returns a snapshot of every captured SendPasswordResetEmail
// call, in call order.
func (m *CapturingMailer) ResetSends() []CapturedEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]CapturedEmail, len(m.resetSends))
	copy(out, m.resetSends)
	return out
}
