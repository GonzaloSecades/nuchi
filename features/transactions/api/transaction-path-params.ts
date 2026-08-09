/**
 * Builds the generated client's path parameters for a single transaction.
 *
 * The guard is not defensive padding. openapi-fetch substitutes path
 * parameters by name and leaves the placeholder untouched when the value is
 * missing, so `{ path: { id: undefined } }` produces a request to
 * `/api/transactions/{id}` — in a browser that resolves and goes out as
 * `%7Bid%7D`, hitting the API as a nonsense id rather than failing where the
 * mistake was made. Failing here turns it into an ordinary rejected mutation
 * with the hook's own fallback toast.
 *
 * Mirrors `accountPathParams` in the accounts lane, which found this first.
 */
export function transactionPathParams(id?: string) {
  if (!id) {
    throw new Error('Transaction id is required');
  }

  return { path: { id } };
}
