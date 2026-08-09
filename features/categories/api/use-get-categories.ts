//hook to get categories to getCategorie endpoint

import { useQuery } from '@tanstack/react-query';

import { apiClient, unwrap } from '@/lib/api/client';

export const useGetCategories = () => {
  const query = useQuery({
    queryKey: ['categories'],
    queryFn: async () => {
      const result = await apiClient.GET('/categories');

      const { data } = unwrap(result, 'categories');
      return data;
    },
  });

  return query;
};
