import { InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<
  (typeof client.api.transactions)[':id']['$delete']
>;

export const useDeleteTransaction = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error>({
    mutationFn: async () => {
      const response = await client.api.transactions[':id']['$delete']({
        param: { id: id },
      });

      if (!response.ok) {
        throw await createApiError(response, 'delete transaction');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Transaction deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['transaction', { id }] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      //TODO: Invalidate Summary
    },
    onError: () => {
      toast.error(`Error to delete transaction`);
    },
  });
  return mutation;
};
