import { isApiError } from '@/lib/api-error';

/**
 * One entry of a bulk operation's `details.fields`, decoded.
 *
 * `index` is the zero-based position of the offending item within the
 * submitted array, or `null` when the problem is with the array itself.
 * `field` is the property that failed, or `null` when the whole item or array
 * is at fault.
 */
export type BulkFieldError = {
  index: number | null;
  field: string | null;
  message: string;
};

/**
 * Matches the indexed paths the API reports for bulk operations.
 *
 * Bulk-create uses `[3].amount` for a problem in row 3 and `$` for one with
 * the array itself — empty, or past the 500-row maximum. Bulk-delete uses
 * `ids` for the array and `ids[2]` for one entry. Both forms are covered here
 * so a caller does not have to know which operation it came from.
 */
const INDEXED_PATH = /^\[(\d+)\](?:\.(.+))?$/;
const IDS_PATH = /^ids(?:\[(\d+)\])?$/;

/**
 * Decodes a bulk operation's per-item validation errors.
 *
 * Returns an empty array for anything that is not a validation error carrying
 * `details.fields`. That is not a defensive nicety: a body the API could not
 * parse at all — malformed JSON, an unknown field, more than one JSON value —
 * has no per-row paths to report and omits `details` entirely, so "no decoded
 * rows" is a real and expected answer that callers must handle with a general
 * message.
 */
export function bulkFieldErrors(error: unknown): BulkFieldError[] {
  if (!isApiError(error)) {
    return [];
  }

  const fields = (
    error.details.errorData as
      | { error?: { details?: { fields?: unknown } } }
      | null
      | undefined
  )?.error?.details?.fields;

  if (!Array.isArray(fields)) {
    return [];
  }

  const decoded: BulkFieldError[] = [];

  for (const entry of fields) {
    if (
      typeof entry !== 'object' ||
      entry === null ||
      typeof (entry as { path?: unknown }).path !== 'string' ||
      typeof (entry as { message?: unknown }).message !== 'string'
    ) {
      continue;
    }

    const { path, message } = entry as { path: string; message: string };

    const indexed = INDEXED_PATH.exec(path);
    if (indexed) {
      decoded.push({
        index: Number(indexed[1]),
        field: indexed[2] ?? null,
        message,
      });
      continue;
    }

    const ids = IDS_PATH.exec(path);
    if (ids) {
      decoded.push({
        index: ids[1] === undefined ? null : Number(ids[1]),
        field: 'ids',
        message,
      });
      continue;
    }

    // `$` is the whole-array case; anything else unrecognized is reported
    // without a position rather than dropped, so the message still reaches
    // the user.
    decoded.push({ index: null, field: path === '$' ? null : path, message });
  }

  return decoded;
}

/**
 * Renders decoded bulk errors as a short, human-readable summary.
 *
 * Rows are numbered from `rowOffset`, which exists because the import screen
 * submits in chunks of 500: an error at index 3 of the second chunk is row
 * 504 of the user's file, and reporting it as row 4 would send them to the
 * wrong line. Callers pass the number of rows already submitted.
 *
 * At most `limit` entries are listed; a long tail is counted rather than
 * printed, since a toast holding 500 lines helps nobody.
 */
export function formatBulkErrorSummary(
  errors: BulkFieldError[],
  { rowOffset = 0, limit = 3 }: { rowOffset?: number; limit?: number } = {}
): string | null {
  if (errors.length === 0) {
    return null;
  }

  const parts = errors.slice(0, limit).map((error) => {
    if (error.index === null) {
      return error.message;
    }
    const row = error.index + rowOffset + 1;
    return error.field === null || error.field === 'ids'
      ? `Row ${row}: ${error.message}`
      : `Row ${row} (${error.field}): ${error.message}`;
  });

  const remaining = errors.length - parts.length;
  if (remaining > 0) {
    parts.push(`and ${remaining} more`);
  }

  return parts.join('; ');
}
