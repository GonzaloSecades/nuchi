import { InferRequestType, InferResponseType } from 'hono';
import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

type ResponseType = InferResponseType<
  (typeof client.api.accounts)['bulk-delete']['$post']
>;
type RequestType = InferRequestType<
  (typeof client.api.accounts)['bulk-delete']['$post']
>['json'];

export const useBulkDeleteAccounts = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const response = await client.api.accounts['bulk-delete']['$post']({
        json,
      });

      if (!response.ok) {
        throw await createApiError(response, 'accounts');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Accounts deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
    onError: () => {
      toast.error(`Error deleting accounts`);
    },
  });
  return mutation;
};
