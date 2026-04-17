'use client';

import { Suspense, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useBulkCreateTransactions } from '@/features/transactions/api/use-bulk-create-transactions';
import { useNewTransaction } from '@/features/transactions/hooks/use-new-transaction';
import { Loader2, Plus } from 'lucide-react';

import { DataTable } from '@/components/data-table';
import { Skeleton } from '@/components/ui/skeleton';
import { useSelectAccount } from '@/features/accounts/hooks/use-select-account';
import { useBulkDeleteTransactions } from '@/features/transactions/api/use-bulk-delete-transactions';
import { useGetTransactions } from '@/features/transactions/api/use-get-transactions';
import { chunkItems } from '@/lib/chunk-items';
import { MAX_BULK_CREATE_TRANSACTIONS } from '@/lib/transaction-limits';
import { toast } from 'sonner';
import { columns } from './columns';
import { ImportCard } from './import-card';
import { UploadButton } from './upload-button';
import type { ImportedTransactionRow } from './import-card';
import type { CSVUploadResults } from './upload-button';

enum VARIANTS {
  LIST = 'LIST',
  IMPORT = 'IMPORT',
}

const INITIAL_IMPORT_RESULTS: CSVUploadResults = {
  data: [],
  errors: [],
  meta: {},
};

const TransactionsPage = () => {
  const [AccountDialog, confirm] = useSelectAccount();

  const [variant, setVariant] = useState<VARIANTS>(VARIANTS.LIST);

  const [importResults, setImportResults] = useState(INITIAL_IMPORT_RESULTS);

  const onUpload = (results: CSVUploadResults) => {
    setImportResults(results);
    setVariant(VARIANTS.IMPORT);
  };

  const onCancelImport = () => {
    setImportResults(INITIAL_IMPORT_RESULTS);
    setVariant(VARIANTS.LIST);
  };

  const newTransaction = useNewTransaction();
  const createTransactions = useBulkCreateTransactions();
  const deleteTransactions = useBulkDeleteTransactions();
  const transactionsQuery = useGetTransactions();
  const transactions = transactionsQuery.data || [];

  const isDisabled =
    transactionsQuery.isLoading || deleteTransactions.isPending;

  const onSubmitImport = async (values: ImportedTransactionRow[]) => {
    const accountId = await confirm();
    if (!accountId) {
      return toast.error('You must select an account to import transactions');
    }

    const data = values.map((value) => ({
      ...value,
      accountId: accountId as string,
    }));

    try {
      for (const chunk of chunkItems(data, MAX_BULK_CREATE_TRANSACTIONS)) {
        await createTransactions.mutateAsync(chunk);
      }

      onCancelImport();
      toast.success('Transactions imported successfully');
    } catch {
      toast.error('Failed to import transactions');
    }
  };

  if (transactionsQuery.isLoading) {
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

  if (variant === VARIANTS.IMPORT) {
    return (
      <>
        <AccountDialog />
        <ImportCard
          data={importResults.data}
          onCancel={onCancelImport}
          onSubmit={onSubmitImport}
        />
      </>
    );
  }

  return (
    <div className="mx-auto -mt-24 w-full max-w-(--breakpoint-2xl) pb-10">
      <Card className="border-none drop-shadow-sm">
        <CardHeader className="gap-y-2 lg:flex lg:flex-row lg:items-center lg:justify-between">
          <CardTitle className="line-clamp-1 text-xl">
            Transactions History
          </CardTitle>
          <div className="flex flex-col items-center gap-x-2 gap-y-2 lg:flex-row">
            <Button
              onClick={newTransaction.onOpen}
              size="sm"
              className="w-full lg:w-auto"
            >
              <Plus className="mr-1 size-4" />
              Add new
            </Button>
            <UploadButton onUpload={onUpload} />
          </div>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={columns}
            data={transactions}
            filterKey="payee"
            onDelete={(row) => {
              const ids = row.map((r) => r.original.id);
              deleteTransactions.mutate({ ids });
            }}
            disabled={isDisabled}
          />
        </CardContent>
      </Card>
    </div>
  );
};

const TransactionsPageWrapper = () => {
  return (
    <Suspense
      fallback={
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
      }
    >
      <TransactionsPage />
    </Suspense>
  );
};

export default TransactionsPageWrapper;
