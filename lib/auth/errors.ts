import { apiErrorCode } from '@/lib/api/client';
import { isApiError } from '@/lib/api-error';

/**
 * Turns a failed auth call into something worth showing a person.
 *
 * Every branch here keys off `apiErrorCode`, never the message text: codes are
 * the contract's stable surface, messages are prose that will eventually be
 * translated. The API's own message is the fallback because it is already
 * written for end users ("Invalid email or password."), and the generic line is
 * only reached for a network failure or an unrecognized shape.
 *
 * `overrides` lets a page phrase a code in its own context — the sign-in page
 * says something different about `EMAIL_NOT_VERIFIED` than the reset page would
 * — without either page having to re-implement the fallback chain.
 */
export function authErrorMessage(
  error: unknown,
  overrides: Record<string, string> = {}
): string {
  const code = apiErrorCode(error);

  if (code !== null && code in overrides) {
    return overrides[code];
  }

  if (isApiError(error) && error.message !== '') {
    return error.message;
  }

  return 'Something went wrong. Please try again.';
}

/** One entry of a `VALIDATION_ERROR` response's `details.fields`. */
type FieldError = { path: string; message: string };

/**
 * Pulls per-field messages out of a `VALIDATION_ERROR` response.
 *
 * The Go handlers report every failing field at once (`details.fields[]`, each
 * with a `path` and a `message`), so a form can attach each one to its own
 * input instead of showing a single summary and making the user guess which
 * box is wrong. Returns an empty array for any other error, and for a
 * validation error whose `details` was omitted — a malformed body carries no
 * field paths at all.
 */
export function validationFieldErrors(error: unknown): FieldError[] {
  if (!isApiError(error)) {
    return [];
  }

  const details = (
    error.details.errorData as
      | { error?: { details?: { fields?: unknown } } }
      | null
      | undefined
  )?.error?.details?.fields;

  if (!Array.isArray(details)) {
    return [];
  }

  return details.filter(
    (field): field is FieldError =>
      typeof field === 'object' &&
      field !== null &&
      typeof (field as FieldError).path === 'string' &&
      typeof (field as FieldError).message === 'string'
  );
}
