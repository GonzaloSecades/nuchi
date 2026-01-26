'use client';

import { z } from 'zod';
import { useNewAccount } from '@/features/accounts/hooks/use-new-account';
import { InsertAccountSchema } from '@/db/schema';
import { AccountForm } from '@/features/accounts/components/account-form';
import { useCreateAccount } from '@/features/accounts/api/use-create-account';

import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';

type FormValues = Pick<z.infer<typeof InsertAccountSchema>, 'name'>;

export const NewAccountSheet = () => {
  const { isOpen, onClose } = useNewAccount();

  const mutation = useCreateAccount();

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values, {
      onSuccess: () => {
        onClose();
      },
    });
  };

  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="space-y-4">
        <SheetHeader>
          <SheetTitle>New Account</SheetTitle>
          <SheetDescription>Create new Account</SheetDescription>
        </SheetHeader>
        <AccountForm
          onSubmit={onSubmit}
          disabled={mutation.isPending}
          defaultValues={{
            name: '',
          }}
        />
      </SheetContent>
    </Sheet>
  );
};
