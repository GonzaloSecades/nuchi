'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useNewAccount } from '@/features/accounts/hooks/use-new-account';
import { Plus } from 'lucide-react';

const AccountsPage = () => {
  const newAccount = useNewAccount();
  return (
    <div className="mx-auto -mt-24 w-full max-w-(--breakpoint-2xl) pb-10">
      <Card className="border-none drop-shadow-sm">
        <CardHeader className="gap-y-2 lg:flex lg:flex-row lg:items-center lg:justify-between">
          <CardTitle className="line-clamp-1 text-xl">Accounts</CardTitle>
          <Button onClick={newAccount.onOpen} size="sm">
            <Plus className="mr-1 size-4" />
            Account
          </Button>
        </CardHeader>
        <CardContent>Accounts Page Content</CardContent>
      </Card>
    </div>
  );
};
export default AccountsPage;
