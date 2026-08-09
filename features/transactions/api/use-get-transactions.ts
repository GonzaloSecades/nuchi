import { useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'next/navigation';

import { calendarDateFromApi } from '@/features/transactions/api/transaction-date';
import { apiClient, unwrap } from '@/lib/api/client';
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
      const result = await apiClient.GET('/transactions', {
        params: {
          // Unset filters are omitted, not sent as empty strings: an
          // explicitly empty `from`/`to`/`accountId` is malformed per the API
          // contract and is rejected with 400 INVALID_QUERY, while omission
          // selects the default range.
          query: omitEmptyQueryParams({ from, to, accountId }),
        },
      });

      const { data } = unwrap(result, 'transactions');
      return data.map((transaction) => ({
        ...transaction,
        amount: convertAmountFromMiliunits(transaction.amount),
        // Normalized here so every consumer sees the calendar date the row
        // actually holds, rather than re-deriving an instant in its own
        // timezone and landing a day earlier.
        date: calendarDateFromApi(transaction.date),
      }));
    },
  });

  return query;
};
