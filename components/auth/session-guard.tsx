'use client';

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { Loader2 } from 'lucide-react';

import { redirectTargetFromLocation } from '@/lib/auth/redirect';
import { useSession } from '@/lib/auth/session';

/**
 * Keeps the dashboard behind a session, client-side.
 *
 * Route protection used to live in Next middleware, where Clerk's cookie was
 * readable. It cannot stay there: the only credential that survives a page load
 * is the refresh token, which is `HttpOnly` *and* scoped to `Path=/api/auth`,
 * so the browser does not send it on a page navigation at all. The middleware
 * sees precisely nothing to authenticate with — this is a consequence of the
 * token design, not a shortcut around it.
 *
 * The trade-off is honest: the dashboard shell can be requested by anyone, so
 * the protection that matters is the API's. Every request carries a Bearer
 * token the server verifies, and RLS backs that per row. This guard is a
 * navigation affordance — it stops signed-out users staring at a screen of
 * failed queries — not the security boundary.
 */
export const SessionGuard = ({ children }: { children: React.ReactNode }) => {
  const { status } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (status !== 'unauthenticated') {
      return;
    }

    // Read the location here rather than through useSearchParams: the redirect
    // only ever runs in the browser, and taking the hook would opt every
    // dashboard page into a Suspense boundary it otherwise does not need.
    const target = redirectTargetFromLocation(window.location);
    router.replace(`/sign-in?redirect=${encodeURIComponent(target)}`);
  }, [status, router]);

  // `loading` covers the bootstrap refresh that every page load starts with.
  // Rendering children during it would fire the dashboard's queries with no
  // token; redirecting during it would bounce signed-in users on every reload.
  if (status !== 'authenticated') {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="size-8 animate-spin text-slate-400" />
      </div>
    );
  }

  return <>{children}</>;
};
