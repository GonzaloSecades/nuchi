import { InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<
  (typeof client.api.accounts)[':id']['$delete']
>;

export const useDeleteAccount = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error>({
    mutationFn: async () => {
      const response = await client.api.accounts[':id']['$delete']({
        param: { id: id },
      });

      if (!response.ok) {
        throw await createApiError(response, 'delete account ');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Account deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['account', { id }] });
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      //TODO: Invalidate Summary
    },
    onError: () => {
      toast.error(`Error to delete account`);
    },
  });
  return mutation;
};
