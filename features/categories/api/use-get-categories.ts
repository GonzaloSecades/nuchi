//hook to get categories to getCategorie endpoint

import { useQuery } from '@tanstack/react-query';

import { createApiError } from '@/lib/api-error';
import { client } from '@/lib/hono';

export const useGetCategories = () => {
  const query = useQuery({
    queryKey: ['categories'],
    queryFn: async () => {
      const response = await client.api.categories.$get();

      if (!response.ok) {
        throw await createApiError(response, 'categories');
      }

      const { data } = await response.json();
      return data;
    },
  });

  return query;
};
