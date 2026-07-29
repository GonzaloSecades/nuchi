/**
 * Builds a query object that omits unset parameters instead of sending them as
 * empty strings.
 *
 * `?from=` is not the same request as omitting `from`. The API's `from`, `to`
 * and `accountId` parameters reference schemas with a minimum length and none
 * sets `allowEmptyValue`, which defaults to `false` in OpenAPI 3.0.3, so an
 * explicitly empty value is malformed and is rejected with
 * `400 INVALID_QUERY`. Only omission selects the default range.
 *
 * Callers therefore have to drop unset filters rather than pass `''`.
 */
export function omitEmptyQueryParams<T extends Record<string, string | undefined>>(
  params: T
): Partial<Record<keyof T, string>> {
  const result: Partial<Record<keyof T, string>> = {};
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') {
      result[key as keyof T] = value;
    }
  }
  return result;
}
