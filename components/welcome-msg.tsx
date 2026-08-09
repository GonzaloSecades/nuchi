'use client';

import { useSession } from '@/lib/auth/session';

export const WelcomeMsg = () => {
  const { user } = useSession();

  // The greeting used Clerk's `firstName`. The API's AuthUser carries only id,
  // email and emailVerified — there is no name to greet by, and inventing one
  // from the email local part reads worse than a plain welcome.
  return (
    <div className="mb-4 space-y-2">
      <h2 className="text-2xl font-medium text-white lg:text-4xl">
        Welcome back 👋
      </h2>
      <p className="text-sm text-[#89b6fd] lg:text-base">
        {user ? user.email : 'Great to see you again!'}
      </p>
    </div>
  );
};
