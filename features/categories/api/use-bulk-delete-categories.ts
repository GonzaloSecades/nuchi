import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';
import type { components } from '@/lib/api/generated/schema';

type ResponseType = components['schemas']['DeletedResourceListResponse'];
type RequestType = components['schemas']['BulkDeleteRequest'];

export const useBulkDeleteCategories = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const result = await apiClient.POST('/categories/bulk-delete', {
        body: json,
      });

      return unwrap(result, 'categories');
    },
    onSuccess: () => {
      toast.success('Categories deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    onError: () => {
      toast.error(`Error deleting categories`);
    },
  });
  return mutation;
};
