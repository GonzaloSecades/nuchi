import { InferRequestType, InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { ApiError, createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<
  (typeof client.api.accounts)[':id']['$patch']
>;
type RequestType = InferRequestType<
  (typeof client.api.accounts)[':id']['$patch']
>['json'];

export const useEditAccount = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const response = await client.api.accounts[':id']['$patch']({
        param: { id: id },
        json,
      });

      if (!response.ok) {
        throw await createApiError(response, 'edit account ');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Account edited successfully');
      queryClient.invalidateQueries({ queryKey: ['account', { id }] });
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      //TODO: Invalidate Summary
    },
    onError: (error) => {
      const apiMessage =
        error instanceof ApiError
          ? (error.details.errorData as { error?: { message?: string } } | null)
              ?.error?.message
          : null;

      toast.error(apiMessage ?? 'Error editing account');
    },
  });
  return mutation;
};
