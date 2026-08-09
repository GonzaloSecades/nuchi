//hook to get accounts to getAccount endpoint

import { useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';

export const useGetAccounts = () => {
  const query = useQuery({
    queryKey: ['accounts'],
    queryFn: async () => {
      const result = await apiClient.GET('/accounts');

      const { data } = unwrap(result, 'accounts');
      return data;
    },
  });

  return query;
};
