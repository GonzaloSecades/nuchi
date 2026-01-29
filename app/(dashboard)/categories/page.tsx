'use client';

import { DataTable } from '@/components/data-table';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Loader2, Plus } from 'lucide-react';

import { useBulkDeleteCategories } from '@/features/categories/api/use-bulk-delete-categories';
import { useGetCategories } from '@/features/categories/api/use-get-categories';
import { useNewCategory } from '@/features/categories/hooks/use-new-category';
import { columns } from './columns';

const CategoriesPage = () => {
  const newCategory = useNewCategory();
  const deleteCategories = useBulkDeleteCategories();
  const categoriesQuery = useGetCategories();
  const categories = categoriesQuery.data || [];

  const isDisabled = categoriesQuery.isLoading || deleteCategories.isPending;

  if (categoriesQuery.isLoading) {
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
          <CardTitle className="line-clamp-1 text-xl">Categories</CardTitle>
          <Button onClick={newCategory.onOpen} size="sm">
            <Plus className="mr-1 size-4" />
            Add new
          </Button>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={columns}
            data={categories}
            filterKey="name"
            onDelete={(row) => {
              const ids = row.map((r) => r.original.id);
              deleteCategories.mutate({ ids });
            }}
            disabled={isDisabled}
          />
        </CardContent>
      </Card>
    </div>
  );
};
export default CategoriesPage;
