import { useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'next/navigation';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';
import { omitEmptyQueryParams } from '@/lib/query-params';
import { convertAmountFromMiliunits } from '@/lib/utils';

export const useGetTransactions = () => {
  const params = useSearchParams();

  const from = params.get('from') || '';
  const to = params.get('to') || '';
  const accountId = params.get('accountId') || '';

  const query = useQuery({
    queryKey: ['transactions', { from, to, accountId }],
    queryFn: async () => {
      const response = await client.api.transactions.$get({
        // Unset filters are omitted, not sent as empty strings: an explicitly
        // empty `from`/`to`/`accountId` is malformed per the API contract and
        // is rejected with 400 INVALID_QUERY, while omission selects the
        // default range.
        query: omitEmptyQueryParams({ from, to, accountId }),
      });

      if (!response.ok) {
        throw await createApiError(response, 'transactions');
      }

      const { data } = await response.json();
      return data.map((transaction) => ({
        ...transaction,
        amount: convertAmountFromMiliunits(transaction.amount),
      }));
    },
  });

  return query;
};
