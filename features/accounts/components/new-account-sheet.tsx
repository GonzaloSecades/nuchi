import { useNewAccount } from '@/features/accounts/hooks/use-new-account';

import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';

export const NewAccountSheet = () => {
  const { isOpen, onClose } = useNewAccount();
  return (
    <Sheet open={isOpen} onOpenChange={onClose}>
      <SheetContent className="space-y-4">
        <SheetHeader>
          <SheetTitle>New Account</SheetTitle>
        </SheetHeader>
        <SheetDescription>
          This is where the new account form will go.
        </SheetDescription>
      </SheetContent>
    </Sheet>
  );
};
