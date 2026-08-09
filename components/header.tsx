import { Suspense } from 'react';

import { UserButton } from '@/components/auth/user-button';
import { Filters } from '@/components/filters';
import { HeaderLogo } from '@/components/header-logo';
import { Navigation } from '@/components/navigation';
import { WelcomeMsg } from '@/components/welcome-msg';

export const Header = () => {
  return (
    <header className="bg-linear-to-b from-blue-700 to-blue-500 px-4 py-8 pb-36 lg:px-14">
      <div className="mx-auto max-w-screen-2xl">
        <div className="mb-14 flex w-full items-center justify-between">
          <div className="flex items-center lg:gap-x-16">
            <HeaderLogo />
            <Navigation />
          </div>
          {/* No loading state to straddle: the guard above this only renders
              the dashboard once the session has resolved. */}
          <UserButton />
        </div>
        <WelcomeMsg />
        <Suspense fallback={null}>
          <Filters />
        </Suspense>
      </div>
    </header>
  );
};
