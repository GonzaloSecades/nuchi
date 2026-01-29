import { InferRequestType, InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<
  (typeof client.api.categories)['bulk-delete']['$post']
>;
type RequestType = InferRequestType<
  (typeof client.api.categories)['bulk-delete']['$post']
>['json'];

export const useBulkDeleteCategories = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const response = await client.api.categories['bulk-delete']['$post']({
        json,
      });

      if (!response.ok) {
        throw await createApiError(response, 'categories');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Categories deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['categories'] });
    },
    onError: () => {
      toast.error(`Error deleting categories`);
    },
  });
  return mutation;
};
