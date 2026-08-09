'use client';

import { useState } from 'react';
import { LogOut } from 'lucide-react';
import { toast } from 'sonner';

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useSession } from '@/lib/auth/session';

/**
 * The account menu, replacing Clerk's `UserButton`.
 *
 * Clerk's widget also covered profile management; nothing here does, because
 * the API has no profile operations to back it. The one action it needs to
 * carry is logout, which now revokes the refresh token server-side rather than
 * only clearing local state.
 */
export const UserButton = () => {
  const { user, logout } = useSession();
  const [signingOut, setSigningOut] = useState(false);

  if (!user) {
    return null;
  }

  const initial = user.email.charAt(0).toUpperCase();

  /**
   * Signs out and leaves the navigation to `SessionGuard`.
   *
   * This deliberately does not redirect. Clearing the session flips the guard
   * to `unauthenticated` while it is still mounted around this header, so it
   * redirects on its own — an explicit `router.replace('/sign-in')` here just
   * raced it and lost, which is how it was found. One owner for signed-out
   * navigation means the destination cannot depend on which of two calls
   * happened to run last.
   */
  const onSignOut = async () => {
    setSigningOut(true);
    const { serverConfirmed } = await logout();

    // `logout` never rejects and always clears local state, so the guard
    // redirects either way. What differs is whether the session is actually
    // over: without server confirmation the refresh cookie is still valid, so
    // a reload would sign the user back in. Saying so beats claiming a clean
    // sign-out that did not happen.
    if (!serverConfirmed) {
      toast.error(
        'Signed out on this device, but the server did not confirm it. Sign out again when you are back online.'
      );
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="Account menu"
        className="flex size-8 shrink-0 items-center justify-center rounded-full bg-white/20 text-sm font-medium text-white transition hover:bg-white/30 focus-visible:ring-2 focus-visible:ring-white focus-visible:outline-none"
      >
        {initial}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className="truncate font-normal">
          <span className="text-muted-foreground block text-xs">
            Signed in as
          </span>
          <span className="truncate">{user.email}</span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem disabled={signingOut} onSelect={onSignOut}>
          <LogOut className="mr-2 size-4" />
          {signingOut ? 'Signing out…' : 'Sign out'}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
