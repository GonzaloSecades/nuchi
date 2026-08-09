import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  toBulkTransactionInput,
  type ImportedTransactionInput,
} from '@/features/transactions/api/transaction-payload';
import { apiClient, unwrap } from '@/lib/api/client';

export const useBulkCreateTransactions = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (rows: ImportedTransactionInput[]) => {
      const result = await apiClient.POST('/transactions/bulk-create', {
        // The contract's bulk-create body is a bare array, not an object
        // wrapping one.
        body: toBulkTransactionInput(rows),
      });

      return unwrap(result, 'transactions');
    },
    onSuccess: () => {
      toast.success('Transactions created successfully');
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    // Deliberately no toast here. The import screen catches this to report
    // which row failed, and it is the only caller; a generic toast from the
    // hook would fire alongside that and bury the useful message.
  });
  return mutation;
};
