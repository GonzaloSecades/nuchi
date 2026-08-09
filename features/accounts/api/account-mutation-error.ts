import { ApiError } from '@/lib/api-error';

export const accountMutationErrorMessage = (
  error: unknown,
  fallback: string
): string => {
  if (error instanceof ApiError) {
    return error.message;
  }

  return fallback;
};
