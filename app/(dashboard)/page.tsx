'use client';

import { Button } from '@/components/ui/button';
import { useNewAccount } from '@/features/accounts/hooks/use-new-account';

export default function Home() {
  const { onOpen } = useNewAccount();
  return (
    <>
      <Button onClick={onOpen}>New Account</Button>
    </>
  );
}
