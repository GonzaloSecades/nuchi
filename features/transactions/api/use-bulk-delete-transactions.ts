import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  bulkFieldErrors,
  formatBulkErrorSummary,
} from '@/features/transactions/api/bulk-errors';
import { apiClient, unwrap } from '@/lib/api/client';

export const useBulkDeleteTransactions = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (body: { ids: string[] }) => {
      const result = await apiClient.POST('/transactions/bulk-delete', {
        body,
      });

      return unwrap(result, 'transactions');
    },
    onSuccess: () => {
      toast.success('Transactions deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    onError: (error) => {
      // Selection-driven, so the user has no row numbers in front of them and
      // an indexed message would mean little. The array-level message is the
      // useful one — notably the 500-id cap, which transactions enforce and
      // accounts and categories deliberately do not.
      const summary = formatBulkErrorSummary(
        bulkFieldErrors(error).filter((field) => field.index === null)
      );
      toast.error(summary ?? 'Error deleting transactions');
    },
  });
  return mutation;
};
