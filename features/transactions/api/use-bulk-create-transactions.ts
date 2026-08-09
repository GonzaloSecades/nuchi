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
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    // Deliberately no toast in this hook. The import screen is its only caller
    // and submits in 500-row chunks, so hook-level success messages would fire
    // once per chunk before the page's single completion toast. The page also
    // catches failures to identify the exact row without a competing generic
    // error message.
  });
  return mutation;
};
