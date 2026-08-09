import { ApiError } from '@/lib/api-error';

export const accountMutationErrorMessage = (
  error: unknown,
  fallback: string
): string => {
  if (error instanceof ApiError) {
    const apiMessage = (
      error.details.errorData as {
        error?: { message?: unknown };
      } | null
    )?.error?.message;

    if (typeof apiMessage === 'string' && apiMessage !== '') {
      return apiMessage;
    }
  }

  return fallback;
};
