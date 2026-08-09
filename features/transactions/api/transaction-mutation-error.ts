import { ApiError } from '@/lib/api-error';

/**
 * Picks the message a failed transaction mutation should put in a toast.
 *
 * Reads the API's *structured* message rather than `error.message`, and that
 * distinction is the whole point. `toApiError` sets `message` to the structured
 * text when there is one, but falls back to `Failed to fetch transactions: 500
 * Internal Server Error` when there is not — which is a string to show an
 * engineer, not a person mid-import. The legacy Hono hooks read
 * `errorData.error.message` and fell back to their own wording, so surfacing
 * the generic form would be a regression in the toast text, not just a style
 * change.
 *
 * Mirrors `accountMutationErrorMessage` in the accounts lane. The duplication
 * is deliberate for now — the two lanes are separate PRs and reaching across
 * them would couple them — but this belongs in `lib/api/` once both have
 * landed.
 */
export function transactionMutationErrorMessage(
  error: unknown,
  fallback: string
): string {
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
}
