import { InferRequestType, InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<
  (typeof client.api.categories)[':id']['$patch']
>;
type RequestType = InferRequestType<
  (typeof client.api.categories)[':id']['$patch']
>['json'];

export const useEditCategory = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const response = await client.api.categories[':id']['$patch']({
        param: { id: id },
        json,
      });

      if (!response.ok) {
        throw await createApiError(response, 'edit category ');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Category edited successfully');
      queryClient.invalidateQueries({ queryKey: ['category', { id }] });
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      //TODO: Invalidate Summary and transactions later
    },
    onError: () => {
      toast.error(`Error editing category`);
    },
  });
  return mutation;
};
