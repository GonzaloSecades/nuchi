import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { transactionPathParams } from '@/features/transactions/api/transaction-path-params';
import { apiClient, unwrap } from '@/lib/api/client';

export const useDeleteTransaction = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async () => {
      const result = await apiClient.DELETE('/transactions/{id}', {
        params: transactionPathParams(id),
      });

      return unwrap(result, 'delete transaction');
    },
    onSuccess: () => {
      toast.success('Transaction deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['transaction', { id }] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    onError: () => {
      toast.error(`Error to delete transaction`);
    },
  });
  return mutation;
};
