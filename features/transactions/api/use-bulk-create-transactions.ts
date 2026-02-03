import { InferRequestType, InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<
  (typeof client.api.transactions)['bulk-create']['$post']
>;
type RequestType = InferRequestType<
  (typeof client.api.transactions)['bulk-create']['$post']
>['json'];

export const useBulkCreateTransactions = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const response = await client.api.transactions['bulk-create']['$post']({
        json,
      });

      if (!response.ok) {
        throw await createApiError(response, 'transactions');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Transactions created successfully');
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
    },
    onError: () => {
      toast.error(`Error creating transactions`);
    },
  });
  return mutation;
};
