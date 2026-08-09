import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';
import type { components } from '@/lib/api/generated/schema';

type ResponseType = components['schemas']['DeletedResourceListResponse'];
type RequestType = components['schemas']['BulkDeleteRequest'];

export const useBulkDeleteAccounts = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const result = await apiClient.POST('/accounts/bulk-delete', {
        body: json,
      });

      return unwrap(result, 'accounts');
    },
    onSuccess: () => {
      toast.success('Accounts deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    onError: () => {
      toast.error(`Error deleting accounts`);
    },
  });
  return mutation;
};
