import { useOpenCategory } from '@/features/categories/hooks/use-open-category';
import { useOpenTransaction } from '@/features/transactions/hooks/use-open-transaction';
import { cn } from '@/lib/utils';
import { TriangleAlert } from 'lucide-react';

type Props = {
  id: string;
  category: string | null;
  categoryId: string | null;
};

export const CategoryColumn = ({ id, category, categoryId }: Props) => {
  const { onOpen: onOpenCategory } = useOpenCategory();
  const { onOpen: onOpenTransaction } = useOpenTransaction();

  const onClick = () => {
    if (categoryId) {
      onOpenCategory(categoryId);
    } else {
      // No category assigned, open transaction instead
      onOpenTransaction(id);
    }
  };
  return (
    <div
      className={cn(
        'flex cursor-pointer items-center hover:underline',
        !category && 'text-rose-500'
      )}
      onClick={onClick}
    >
      {!category && <TriangleAlert />}
      {category || 'Uncategorized'}
    </div>
  );
};
