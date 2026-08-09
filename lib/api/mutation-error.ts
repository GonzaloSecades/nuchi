import { ApiError } from '@/lib/api-error';

type ApiErrorEnvelope = {
  error?: {
    code?: unknown;
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
  fallback: string,
  overrides: Record<string, string> = {}
): string {
  if (error instanceof ApiError) {
    const envelope = error.details.errorData as ApiErrorEnvelope | null;
    const code = envelope?.error?.code;

    if (typeof code === 'string' && Object.hasOwn(overrides, code)) {
      return overrides[code];
    }

    const apiMessage = envelope?.error?.message;
    if (typeof apiMessage === 'string' && apiMessage !== '') {
      return apiMessage;
    }
  }

  return fallback;
}
