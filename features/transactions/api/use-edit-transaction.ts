import { InferRequestType, InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { ApiError, createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<
  (typeof client.api.transactions)[':id']['$patch']
>;
type RequestType = InferRequestType<
  (typeof client.api.transactions)[':id']['$patch']
>['json'];

export const useEditTransaction = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const response = await client.api.transactions[':id']['$patch']({
        param: { id: id },
        json,
      });

      if (!response.ok) {
        throw await createApiError(response, 'edit transaction ');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Transaction edited successfully');
      queryClient.invalidateQueries({ queryKey: ['transaction', { id }] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] });
    },
    onError: (error) => {
      const apiMessage =
        error instanceof ApiError
          ? (error.details.errorData as { error?: { message?: string } } | null)
              ?.error?.message
          : null;

      toast.error(apiMessage ?? 'Error editing transaction');
    },
  });
  return mutation;
};
