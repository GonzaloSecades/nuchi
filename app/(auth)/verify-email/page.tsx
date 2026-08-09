'use client';

import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { CircleCheck, CircleX, Loader2 } from 'lucide-react';

import { AuthHeader } from '@/components/auth/auth-header';
import { Button } from '@/components/ui/button';
import { authErrorMessage } from '@/lib/auth/errors';
import { verifyEmailToken } from '@/lib/auth/session-requests';

type Outcome =
  | { state: 'verifying' }
  | { state: 'verified'; message: string }
  | { state: 'failed'; message: string; retryable: boolean };

/**
 * Consumes the `?token=` an emailed verification link carries.
 *
 * The URL is built by the Go mailer (`verifyEmailPath = "/verify-email"`), so
 * this route's path is fixed by the backend, not chosen here. Verification is
 * submitted automatically on arrival: the user already expressed intent by
 * clicking the link in their inbox, and a second "confirm" button would only
 * add a step that can be abandoned.
 */
const VerifyEmail = () => {
  const searchParams = useSearchParams();
  const token = searchParams.get('token');
  const [attempt, setAttempt] = useState(0);
  /**
   * The settled outcome, tagged with the token it belongs to.
   *
   * Tagging is what makes "verifying" a *derived* state rather than something
   * an effect has to set: any token without a matching settled result is still
   * being checked. It also discards a late response from a token the user has
   * navigated away from, since that result no longer matches the current token
   * and simply never renders.
   */
  const [settled, setSettled] = useState<{
    token: string;
    outcome: Exclude<Outcome, { state: 'verifying' }>;
  } | null>(null);

  const outcome: Outcome =
    settled !== null && settled.token === token
      ? settled.outcome
      : { state: 'verifying' };

  useEffect(() => {
    // A link with no token is decided at render time below, not here — there
    // is nothing to ask the server about.
    if (!token) {
      return;
    }

    /**
     * Set when this effect is superseded, so a slow request cannot write state
     * on the way out.
     *
     * The token tag alone is not enough. It stops a stale outcome being
     * *rendered*, but not from being *stored*: if the user moves from token A
     * to token B, B settles first and A arrives late, the late write replaces
     * B's result with A's, no tag matches the current token any more, and the
     * page falls back to "Verifying" and stays there.
     */
    let active = true;

    // Memoized per token. A repeated effect for the *same* token replays the
    // first result instead of resubmitting — the token is single-use, so a
    // second submission would report INVALID_TOKEN for a verification that had
    // in fact just succeeded. A *different* token issues a real request, which
    // a boolean "already submitted" flag could not do: Next keeps client state
    // when only the search parameter changes, so the second link would sit on
    // "Verifying" forever.
    verifyEmailToken(token).then((result) => {
      if (!active) {
        return;
      }

      setSettled({
        token,
        outcome:
          result.status === 'verified'
            ? { state: 'verified', message: result.message }
            : {
                state: 'failed',
                retryable: result.retryable,
                message: authErrorMessage(result.error, {
                  INVALID_TOKEN:
                    'This verification link is invalid or has expired. Sign up again with the same email to get a new one.',
                }),
              },
      });
    });

    return () => {
      active = false;
    };
  }, [attempt, token]);

  if (!token) {
    return (
      <div className="space-y-6 text-center">
        <CircleX className="mx-auto size-12 text-rose-500" />
        <AuthHeader
          title="Link incomplete"
          description="This verification link is missing its token. Open the link from your email directly, without editing it."
        />
        <Button asChild className="w-full">
          <Link href="/sign-up">Back to sign up</Link>
        </Button>
      </div>
    );
  }

  if (outcome.state === 'verifying') {
    return (
      <div className="space-y-6 text-center">
        <Loader2 className="mx-auto size-12 animate-spin text-blue-500" />
        <AuthHeader
          title="Verifying your email"
          description="This will only take a moment."
        />
      </div>
    );
  }

  const verified = outcome.state === 'verified';

  return (
    <div className="space-y-6 text-center">
      {verified ? (
        <CircleCheck className="mx-auto size-12 text-emerald-500" />
      ) : (
        <CircleX className="mx-auto size-12 text-rose-500" />
      )}
      <AuthHeader
        title={verified ? 'Email verified' : 'Verification failed'}
        description={outcome.message}
      />
      {!verified && outcome.state === 'failed' && outcome.retryable ? (
        <Button
          className="w-full"
          onClick={() => {
            setSettled(null);
            setAttempt((value) => value + 1);
          }}
        >
          Try again
        </Button>
      ) : (
        <Button asChild className="w-full">
          <Link href={verified ? '/sign-in' : '/sign-up'}>
            {verified ? 'Continue to sign in' : 'Back to sign up'}
          </Link>
        </Button>
      )}
    </div>
  );
};

export default function VerifyEmailPage() {
  return (
    <Suspense
      fallback={
        <Loader2 className="text-muted-foreground mx-auto animate-spin" />
      }
    >
      <VerifyEmail />
    </Suspense>
  );
}
