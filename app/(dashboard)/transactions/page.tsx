'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useNewTransaction } from '@/features/transactions/hooks/use-new-transaction';
import { Loader2, Plus } from 'lucide-react';

import { DataTable } from '@/components/data-table';
import { Skeleton } from '@/components/ui/skeleton';
import { useBulkDeleteAccounts } from '@/features/accounts/api/use-bulk-delete-accounts';
import { useGetAccounts } from '@/features/accounts/api/use-get-accounts';
import { columns } from './columns';

const TransactionsPage = () => {
  const newTransaction = useNewTransaction();
  const deleteAccounts = useBulkDeleteAccounts();
  const accountsQuery = useGetAccounts();
  const accounts = accountsQuery.data || [];

  const isDisabled = accountsQuery.isLoading || deleteAccounts.isPending;

  if (accountsQuery.isLoading) {
    return (
      <div className="mx-auto -mt-24 w-full max-w-(--breakpoint-2xl) pb-10">
        <Card className="border-none drop-shadow-sm">
          <CardHeader>
            <Skeleton className="h-8 w-48" />
          </CardHeader>
          <CardContent className="flex h-125 w-full items-center justify-center">
            <Loader2 className="size-6 animate-spin text-slate-300" />
          </CardContent>
        </Card>
      </div>
    );
  }
  return (
    <div className="mx-auto -mt-24 w-full max-w-(--breakpoint-2xl) pb-10">
      <Card className="border-none drop-shadow-sm">
        <CardHeader className="gap-y-2 lg:flex lg:flex-row lg:items-center lg:justify-between">
          <CardTitle className="line-clamp-1 text-xl">
            Transactions History
          </CardTitle>
          <Button onClick={newTransaction.onOpen} size="sm">
            <Plus className="mr-1 size-4" />
            Add new
          </Button>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={columns}
            data={accounts}
            filterKey="name"
            onDelete={(row) => {
              const ids = row.map((r) => r.original.id);
              deleteAccounts.mutate({ ids });
            }}
            disabled={isDisabled}
          />
        </CardContent>
      </Card>
    </div>
  );
};
export default TransactionsPage;
