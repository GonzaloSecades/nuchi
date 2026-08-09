import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  toTransactionInput,
  type TransactionFormValues,
} from '@/features/transactions/api/transaction-payload';
import { transactionMutationErrorMessage } from '@/features/transactions/api/transaction-mutation-error';
import { transactionPathParams } from '@/features/transactions/api/transaction-path-params';
import { apiClient, unwrap } from '@/lib/api/client';

export const useEditTransaction = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (values: TransactionFormValues) => {
      const result = await apiClient.PATCH('/transactions/{id}', {
        params: transactionPathParams(id),
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
        transactionMutationErrorMessage(error, 'Error editing transaction')
      );
    },
  });
  return mutation;
};
