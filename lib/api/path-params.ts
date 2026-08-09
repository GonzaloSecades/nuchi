/**
 * Builds a generated-client path parameter and rejects a missing id.
 *
 * openapi-fetch leaves an undefined placeholder in the URL, which would send
 * a request such as `/api/accounts/%7Bid%7D` instead of failing at the caller.
 */
export function requiredPathParams(resourceName: string, id?: string) {
  if (!id) {
    throw new Error(`${resourceName} id is required`);
  }

  return { path: { id } };
}
