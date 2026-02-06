import { useQuery } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

export const useGetTransaction = (id?: string) => {
  const query = useQuery({
    enabled: !!id,
    queryKey: ['transaction', { id }],
    queryFn: async () => {
      const response = await client.api.transactions[':id'].$get({
        param: { id },
      });

      if (!response.ok) {
        throw await createApiError(response, 'getSingleTransaction');
      }

      const { data } = await response.json();
      return data;
    },
  });

  return query;
};
