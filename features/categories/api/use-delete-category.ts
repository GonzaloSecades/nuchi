import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';
import type { components } from '@/lib/api/generated/schema';
import { requiredPathParams } from '@/lib/api/path-params';

type ResponseType = components['schemas']['DeletedResourceResponse'];

export const useDeleteCategory = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error>({
    mutationFn: async () => {
      const result = await apiClient.DELETE('/categories/{id}', {
        params: requiredPathParams('Category', id),
      });

      return unwrap(result, 'delete category');
    },
    onSuccess: () => {
      toast.success('Category deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['category', { id }] });
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    onError: () => {
      toast.error(`Error deleting category`);
    },
  });
  return mutation;
};
