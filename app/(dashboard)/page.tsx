'use client';

import { Button } from '@/components/ui/button';
import { useNewAccount } from '@/features/accounts/hooks/use-new-account';

export default function Home() {
  const { onOpen } = useNewAccount();

  const handleClick = () => {
    onOpen();
  };

  return (
    <>
      <Button onClick={handleClick}>New Account</Button>
    </>
  );
}
