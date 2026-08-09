import { ApiError } from '@/lib/api-error';

type ApiErrorEnvelope = {
  error?: {
    message?: unknown;
  };
};

/**
 * Picks the user-facing message for a failed resource mutation.
 *
 * The structured API message is deliberately read from `errorData` rather
 * than `error.message`: an `ApiError` without a response envelope carries a
 * technical message such as "Failed to fetch accounts: 500 Internal Server
 * Error", which must not reach a toast. Callers keep control of that case with
 * their existing fallback wording.
 */
export function mutationErrorMessage(
  error: unknown,
  fallback: string
): string {
  if (error instanceof ApiError) {
    const envelope = error.details.errorData as ApiErrorEnvelope | null;
    const apiMessage = envelope?.error?.message;
    if (typeof apiMessage === 'string' && apiMessage !== '') {
      return apiMessage;
    }
  }

  return fallback;
}
