import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  toTransactionInput,
  type TransactionFormValues,
} from '@/features/transactions/api/transaction-payload';
import { apiClient, unwrap } from '@/lib/api/client';
import { mutationErrorMessage } from '@/lib/api/mutation-error';

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
      toast.error(mutationErrorMessage(error, 'Error creating transaction'));
    },
  });
  return mutation;
};
