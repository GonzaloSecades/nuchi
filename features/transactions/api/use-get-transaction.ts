import { useQuery } from '@tanstack/react-query';

import { transactionPathParams } from '@/features/transactions/api/transaction-path-params';
import { apiClient, unwrap } from '@/lib/api/client';
import { convertAmountFromMiliunits } from '@/lib/utils';

export const useGetTransaction = (id?: string) => {
  const query = useQuery({
    enabled: !!id,
    queryKey: ['transaction', { id }],
    queryFn: async () => {
      const result = await apiClient.GET('/transactions/{id}', {
        params: transactionPathParams(id),
      });

      const { data } = unwrap(result, 'getSingleTransaction');
      return {
        ...data,
        amount: convertAmountFromMiliunits(data.amount),
      };
    },
  });

  return query;
};
