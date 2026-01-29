'use client';

import { useConfirm } from '@/hooks/use-confirm';

import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Edit, MoreHorizontal, Trash } from 'lucide-react';

import { useDeleteCategory } from '@/features/categories/api/use-delete-category';
import { useOpenCategory } from '@/features/categories/hooks/use-open-category';

type Props = {
  id: string;
};

export const Actions = ({ id }: Props) => {
  const [ConfirmDialog, confirm] = useConfirm(
    'Are you sure?',
    'You are about to delete this category. This action cannot be undone.'
  );
  const deleteMutation = useDeleteCategory(id);
  const { onOpen } = useOpenCategory();
  const handleDelete = async () => {
    const ok = await confirm();
    if (ok) {
      deleteMutation.mutate(undefined, {});
    }
  };
  return (
    <>
      <ConfirmDialog />
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="size-8 p-0">
            <MoreHorizontal className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem
            disabled={deleteMutation.isPending}
            onClick={() => {
              onOpen(id);
            }}
          >
            <Edit className="mr-2 size-4" />
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem
            disabled={deleteMutation.isPending}
            onClick={() => {
              handleDelete();
            }}
          >
            <Trash className="mr-2 size-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
};
