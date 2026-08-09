import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  toTransactionInput,
  type TransactionFormValues,
} from '@/features/transactions/api/transaction-payload';
import { apiClient, unwrap } from '@/lib/api/client';
import { ApiError } from '@/lib/api-error';

export const useEditTransaction = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (values: TransactionFormValues) => {
      const result = await apiClient.PATCH('/transactions/{id}', {
        // The sheet only mounts this with an id; the generated path type
        // requires a string.
        params: { path: { id: id as string } },
        body: toTransactionInput(values),
      });

      return unwrap(result, 'edit transaction');
    },
    onSuccess: () => {
      toast.success('Transaction edited successfully');
      queryClient.invalidateQueries({ queryKey: ['transaction', { id }] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    onError: (error) => {
      toast.error(
        error instanceof ApiError ? error.message : 'Error editing transaction'
      );
    },
  });
  return mutation;
};
