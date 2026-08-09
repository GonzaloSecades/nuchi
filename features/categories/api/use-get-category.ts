import { useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';

import { categoryPathParams } from './category-path-params';

export const useGetCategory = (id?: string) => {
  const query = useQuery({
    enabled: !!id,
    queryKey: ['category', { id }],
    queryFn: async () => {
      const result = await apiClient.GET('/categories/{id}', {
        params: categoryPathParams(id),
      });

      const { data } = unwrap(result, 'getSingleCategory');
      return data;
    },
  });

  return query;
};
