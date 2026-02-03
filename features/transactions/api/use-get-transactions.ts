//hook to get accounts to getAccount endpoint

import { useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'next/navigation';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

export const useGetTransactions = () => {
  const params = useSearchParams();

  const from = params.get('from') || '';
  const to = params.get('to') || '';
  const accountId = params.get('accountId') || '';

  const query = useQuery({
    //TODO: Check if params ar needed in the key;

    queryKey: ['transactions'],
    queryFn: async () => {
      const response = await client.api.transactions.$get({
        query: {
          from,
          to,
          accountId,
        },
      });

      if (!response.ok) {
        throw await createApiError(response, 'transactions');
      }

      const { data } = await response.json();
      return data;
    },
  });

  return query;
};
