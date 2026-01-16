//hook to get account to getAccount endpoint

import { useQuery } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

export const useGetAccounts = () => {
  const query = useQuery({
    queryKey: ['accounts'],
    queryFn: async (c) => {
      console.log(c);
      const response = await client.api.accounts.$get();

      if (!response.ok) {
        throw await createApiError(response, 'accounts');
      }

      const { data } = await response.json();
      return data;
    },
  });

  return query;
};
