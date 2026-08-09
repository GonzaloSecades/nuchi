import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';
import type { components } from '@/lib/api/generated/schema';
import { mutationErrorMessage } from '@/lib/api/mutation-error';
import { requiredPathParams } from '@/lib/api/path-params';

type ResponseType = components['schemas']['AccountResponse'];
type RequestType = components['schemas']['AccountInput'];

export const useEditAccount = (id?: string) => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const result = await apiClient.PATCH('/accounts/{id}', {
        params: requiredPathParams('Account', id),
        body: json,
      });

      return unwrap(result, 'edit account');
    },
    onSuccess: () => {
      toast.success('Account edited successfully');
      queryClient.invalidateQueries({ queryKey: ['account', { id }] });
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      queryClient.invalidateQueries({ queryKey: ['summary'] }); //jic
    },
    onError: (error) => {
      toast.error(mutationErrorMessage(error, 'Error editing account'));
    },
  });
  return mutation;
};
