import { InferRequestType, InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<
  (typeof client.api.transactions)['bulk-delete']['$post']
>;
type RequestType = InferRequestType<
  (typeof client.api.transactions)['bulk-delete']['$post']
>['json'];

export const useBulkDeleteTransactions = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const response = await client.api.transactions['bulk-delete']['$post']({
        json,
      });

      if (!response.ok) {
        throw await createApiError(response, 'transactions');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Transactions deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
    },
    onError: () => {
      toast.error(`Error deleting transactions`);
    },
  });
  return mutation;
};
