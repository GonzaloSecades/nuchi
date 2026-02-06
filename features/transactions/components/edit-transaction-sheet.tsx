'use client';

import { InsertTransactionSchema } from '@/db/schema';
import { useDeleteTransaction } from '@/features/transactions/api/use-delete-transaction';
import { useEditTransaction } from '@/features/transactions/api/use-edit-transaction';
import { useGetTransaction } from '@/features/transactions/api/use-get-transaction';
import { TransactionForm } from '@/features/transactions/components/transaction-form';
import { useOpenTransaction } from '@/features/transactions/hooks/use-open-transaction';
import { useConfirm } from '@/hooks/use-confirm';
import { z } from 'zod';

import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { useCreateAccount } from '@/features/accounts/api/use-create-account';
import { useGetAccounts } from '@/features/accounts/api/use-get-accounts';
import { useCreateCategory } from '@/features/categories/api/use-create-category';
import { useGetCategories } from '@/features/categories/api/use-get-categories';
import { Loader2 } from 'lucide-react';

type FormValues = Omit<z.infer<typeof InsertTransactionSchema>, 'id'>;

export const EditTransactionSheet = () => {
  const { isOpen, onClose, id } = useOpenTransaction();

  const [ConfirmDialog, confirm] = useConfirm(
    'Are you sure?',
    'You are about to delete this transaction. This action cannot be undone.'
  );

  // Transactions
  const transactionQuery = useGetTransaction(id);
  const editMutation = useEditTransaction(id);
  const deleteMutation = useDeleteTransaction(id);

  //Categories
  const categoryQuery = useGetCategories();
  const categoryMutation = useCreateCategory();
  const onCreateCategory = (name: string) =>
    categoryMutation.mutate({
      name,
    });
  const categoryOptions = (categoryQuery.data ?? []).map((category) => ({
    label: category.name,
    value: category.id,
  }));

  //Accounts
  const accountQuery = useGetAccounts();
  const accountMutation = useCreateAccount();
  const onCreateAccount = (name: string) =>
    accountMutation.mutate({
      name,
    });
  const accountOptions = (accountQuery.data ?? []).map((account) => ({
    label: account.name,
    value: account.id,
  }));

  const isPending =
    editMutation.isPending ||
    deleteMutation.isPending ||
    transactionQuery.isLoading ||
    categoryMutation.isPending ||
    accountMutation.isPending;

  const isLoading =
    transactionQuery.isLoading ||
    categoryQuery.isLoading ||
    accountQuery.isLoading;

  const onSubmit = (values: FormValues) => {
    editMutation.mutate(values, {
      onSuccess: () => {
        onClose();
      },
    });
  };

  const onDelete = async () => {
    const ok = await confirm();

    if (ok) {
      deleteMutation.mutate(undefined, {
        onSuccess: () => {
          onClose();
        },
      });
    }
  };

  const defaultValues = transactionQuery.data
    ? {
        accountId: transactionQuery.data.accountId,
        categoryId: transactionQuery.data.categoryId,
        amount: transactionQuery.data.amount.toString(),
        date: transactionQuery.data.date
          ? new Date(transactionQuery.data.date)
          : new Date(),
        payee: transactionQuery.data.payee,
        notes: transactionQuery.data.notes || '',
      }
    : {
        accountId: '',
        categoryId: '',
        amount: '',
        date: new Date(),
        payee: '',
        notes: '',
      };

  return (
    <>
      <ConfirmDialog />
      <Sheet open={isOpen} onOpenChange={onClose}>
        <SheetContent className="space-y-4">
          <SheetHeader>
            <SheetTitle>Edit transaction</SheetTitle>
            <SheetDescription>Edit an existing transaction</SheetDescription>
          </SheetHeader>
          {isLoading ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <Loader2 className="text-muted-foreground size-4 animate-spin" />
            </div>
          ) : (
            <TransactionForm
              onDelete={onDelete}
              defaultValues={defaultValues}
              onSubmit={onSubmit}
              disabled={isPending}
              categoryOptions={categoryOptions}
              accountOptions={accountOptions}
              onCreateCategory={onCreateCategory}
              onCreateAccount={onCreateAccount}
            />
          )}
        </SheetContent>
      </Sheet>
    </>
  );
};
