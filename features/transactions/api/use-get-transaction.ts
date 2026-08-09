import { useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';
import { convertAmountFromMiliunits } from '@/lib/utils';

export const useGetTransaction = (id?: string) => {
  const query = useQuery({
    enabled: !!id,
    queryKey: ['transaction', { id }],
    queryFn: async () => {
      const result = await apiClient.GET('/transactions/{id}', {
        // `enabled` keeps this from running without an id, which is why the
        // assertion is safe; the generated path type requires a string.
        params: { path: { id: id as string } },
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
