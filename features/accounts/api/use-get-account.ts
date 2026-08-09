//hook to get a single account to getAccount endpoint

import { useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';

import { accountPathParams } from './account-path-params';

export const useGetAccount = (id?: string) => {
  const query = useQuery({
    enabled: !!id,
    queryKey: ['accounts', { id }],
    queryFn: async () => {
      const result = await apiClient.GET('/accounts/{id}', {
        params: accountPathParams(id),
      });

      const { data } = unwrap(result, 'getSingleAccount');
      return data;
    },
  });

  return query;
};
