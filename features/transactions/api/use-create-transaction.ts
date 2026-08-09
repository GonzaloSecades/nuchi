import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  toTransactionInput,
  type TransactionFormValues,
} from '@/features/transactions/api/transaction-payload';
import { apiClient, unwrap } from '@/lib/api/client';
import { ApiError } from '@/lib/api-error';

export const useCreateTransaction = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (values: TransactionFormValues) => {
      const result = await apiClient.POST('/transactions', {
        body: toTransactionInput(values),
      });

      return unwrap(result, 'transactions');
    },
    onSuccess: () => {
      toast.success('Transaction created successfully');
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    onError: (error) => {
      // `toApiError` already prefers the API's own message, so this reads it
      // straight off the error rather than digging into `errorData` as the
      // Hono version had to.
      toast.error(
        error instanceof ApiError ? error.message : 'Error creating transaction'
      );
    },
  });
  return mutation;
};
