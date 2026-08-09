import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';
import type { components } from '@/lib/api/generated/schema';

import { categoryPathParams } from './category-path-params';

type ResponseType = components['schemas']['CategoryResponse'];
type RequestType = components['schemas']['CategoryInput'];

export const useEditCategory = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const result = await apiClient.PATCH('/categories/{id}', {
        params: categoryPathParams(id),
        body: json,
      });

      return unwrap(result, 'edit category');
    },
    onSuccess: () => {
      toast.success('Category edited successfully');
      queryClient.invalidateQueries({ queryKey: ['category', { id }] });
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    onError: () => {
      toast.error(`Error editing category`);
    },
  });
  return mutation;
};
