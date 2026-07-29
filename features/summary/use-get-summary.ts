import { useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'next/navigation';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';
import { omitEmptyQueryParams } from '@/lib/query-params';
import { convertAmountFromMiliunits } from '@/lib/utils';

export const useGetSummary = () => {
  const params = useSearchParams();

  const from = params.get('from') || '';
  const to = params.get('to') || '';
  const accountId = params.get('accountId') || '';

  const query = useQuery({
    queryKey: ['summary', { from, to, accountId }],
    queryFn: async () => {
      const response = await client.api.summary.$get({
        // Unset filters are omitted, not sent as empty strings: an explicitly
        // empty `from`/`to`/`accountId` is malformed per the API contract and
        // is rejected with 400 INVALID_QUERY, while omission selects the
        // default range.
        query: omitEmptyQueryParams({ from, to, accountId }),
      });

      if (!response.ok) {
        throw await createApiError(response, 'summary');
      }

      const { data } = await response.json();
      return {
        ...data,
        incomeAmount: convertAmountFromMiliunits(data.incomeAmount),
        expensesAmount: convertAmountFromMiliunits(data.expensesAmount),
        remainingAmount: convertAmountFromMiliunits(data.remainingAmount),
        categories: data.categories.map((category) => ({
          ...category,
          value: convertAmountFromMiliunits(category.value),
        })),
        days: data.days.map((day) => ({
          ...day,
          income: convertAmountFromMiliunits(day.income),
          expenses: convertAmountFromMiliunits(day.expenses),
        })),
      };
    },
  });

  return query;
};
