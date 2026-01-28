import { toast } from 'sonner';
import { InferRequestType, InferResponseType } from 'hono';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { client } from '@/lib/hono';
import { createApiError } from '@/lib/api-error';

type ResponseType = InferResponseType<typeof client.api.accounts.$post>;
type RequestType = InferRequestType<typeof client.api.accounts.$post>['json'];

export const useCreateAccount = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const response = await client.api.accounts.$post({ json });

      if (!response.ok) {
        throw await createApiError(response, 'accounts');
      }
      return await response.json();
    },
    onSuccess: () => {
      toast.success('Account created successfully');
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
    onError: () => {
      toast.error(`Error creating account`);
    },
  });
  return mutation;
};
