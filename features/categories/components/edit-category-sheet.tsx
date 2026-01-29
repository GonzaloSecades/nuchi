'use client';

import { InsertCategorySchema } from '@/db/schema';

import { useDeleteCategory } from '@/features/categories/api/use-delete-category';
import { useEditCategory } from '@/features/categories/api/use-edit-category';
import { useGetCategory } from '@/features/categories/api/use-get-category';
import { CategoryForm } from '@/features/categories/components/category-form';
import { useOpenCategory } from '@/features/categories/hooks/use-open-category';
import { useConfirm } from '@/hooks/use-confirm';
import { z } from 'zod';

import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Loader2 } from 'lucide-react';

type FormValues = Pick<z.infer<typeof InsertCategorySchema>, 'name'>;

export const EditCategorySheet = () => {
  const { isOpen, onClose, id } = useOpenCategory();

  const [ConfirmDialog, confirm] = useConfirm(
    'Are you sure?',
    'You are about to delete this category. This action cannot be undone.'
  );

  const categoryQuery = useGetCategory(id);
  const editMutation = useEditCategory(id);
  const deleteMutation = useDeleteCategory(id);

  const isPending = editMutation.isPending || deleteMutation.isPending;

  const isLoading = categoryQuery.isLoading;

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

  const defaultValues = categoryQuery.data
    ? {
        name: categoryQuery.data.name,
      }
    : {
        name: '',
      };

  return (
    <>
      <ConfirmDialog />
      <Sheet open={isOpen} onOpenChange={onClose}>
        <SheetContent className="space-y-4">
          <SheetHeader>
            <SheetTitle>Edit category</SheetTitle>
            <SheetDescription>Edit an existing category</SheetDescription>
          </SheetHeader>
          {isLoading ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <Loader2 className="text-muted-foreground size-4 animate-spin" />
            </div>
          ) : (
            <CategoryForm
              id={id}
              onSubmit={onSubmit}
              disabled={isPending}
              defaultValues={defaultValues}
              onDelete={onDelete}
            />
          )}
        </SheetContent>
      </Sheet>
    </>
  );
};
