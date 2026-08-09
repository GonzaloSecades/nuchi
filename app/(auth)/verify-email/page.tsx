'use client';

import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useRef, useState } from 'react';
import { CircleCheck, CircleX, Loader2 } from 'lucide-react';

import { AuthHeader } from '@/components/auth/auth-header';
import { Button } from '@/components/ui/button';
import { apiClient, unwrap } from '@/lib/api/client';
import { authErrorMessage } from '@/lib/auth/errors';

type Outcome =
  | { state: 'verifying' }
  | { state: 'verified'; message: string }
  | { state: 'failed'; message: string };

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
  const [outcome, setOutcome] = useState<Outcome>({ state: 'verifying' });

  /**
   * Guards against a second submission of a single-use token.
   *
   * React runs effects twice in development Strict Mode. The token is consumed
   * by the first call, so without this the second would consume nothing, come
   * back `INVALID_TOKEN`, and show a failure for a verification that had in
   * fact just succeeded.
   */
  const submitted = useRef(false);

  useEffect(() => {
    // A link with no token is decided at render time below, not here — there
    // is nothing to ask the server about.
    if (!token || submitted.current) {
      return;
    }
    submitted.current = true;

    let cancelled = false;

    (async () => {
      try {
        const result = await apiClient.POST('/auth/verify-email', {
          body: { token },
        });
        const { message } = unwrap(result, 'email verification');
        if (!cancelled) {
          setOutcome({ state: 'verified', message });
        }
      } catch (error) {
        if (!cancelled) {
          setOutcome({
            state: 'failed',
            message: authErrorMessage(error, {
              INVALID_TOKEN:
                'This verification link is invalid or has expired. Sign up again with the same email to get a new one.',
            }),
          });
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [token]);

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
      <Button asChild className="w-full">
        <Link href={verified ? '/sign-in' : '/sign-up'}>
          {verified ? 'Continue to sign in' : 'Back to sign up'}
        </Link>
      </Button>
    </div>
  );
};

export default function VerifyEmailPage() {
  return (
    <Suspense
      fallback={<Loader2 className="text-muted-foreground mx-auto animate-spin" />}
    >
      <VerifyEmail />
    </Suspense>
  );
}
