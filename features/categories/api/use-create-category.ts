import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';
import type { components } from '@/lib/api/generated/schema';
import { mutationErrorMessage } from '@/lib/api/mutation-error';

type ResponseType = components['schemas']['CategoryResponse'];
type RequestType = components['schemas']['CategoryInput'];

export const useCreateCategory = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const result = await apiClient.POST('/categories', { body: json });

      return unwrap(result, 'categories');
    },
    onSuccess: () => {
      toast.success('Category created successfully');
      queryClient.invalidateQueries({ queryKey: ['categories'] });
    },
    onError: (error) => {
      toast.error(mutationErrorMessage(error, 'Error creating category'));
    },
  });
  return mutation;
};
