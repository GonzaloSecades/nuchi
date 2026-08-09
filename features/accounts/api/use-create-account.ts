import { toast } from 'sonner';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';
import type { components } from '@/lib/api/generated/schema';
import { mutationErrorMessage } from '@/lib/api/mutation-error';

type ResponseType = components['schemas']['AccountResponse'];
type RequestType = components['schemas']['AccountInput'];

export const useCreateAccount = () => {
  const queryClient = useQueryClient();

  const mutation = useMutation<ResponseType, Error, RequestType>({
    mutationFn: async (json) => {
      const result = await apiClient.POST('/accounts', { body: json });

      return unwrap(result, 'accounts');
    },
    onSuccess: () => {
      toast.success('Account created successfully');
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
    onError: (error) => {
      toast.error(mutationErrorMessage(error, 'Error creating account'));
    },
  });
  return mutation;
};
