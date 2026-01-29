import { InferRequestType, InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { ApiError, createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<typeof client.api.categories.$post>;
type RequestType = InferRequestType<typeof client.api.categories.$post>['json'];

export const useCreateCategory = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const response = await client.api.categories.$post({ json });

      if (!response.ok) {
        throw await createApiError(response, 'categories');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Category created successfully');
      queryClient.invalidateQueries({ queryKey: ['categories'] });
    },
    onError: (error) => {
      const apiMessage =
        error instanceof ApiError
          ? (error.details.errorData as { error?: { message?: string } } | null)
              ?.error?.message
          : null;

      toast.error(apiMessage ?? 'Error creating category');
    },
  });
  return mutation;
};
