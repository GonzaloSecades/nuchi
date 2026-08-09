/**
 * The calendar-date boundary for transactions.
 *
 * A transaction date is a **calendar date** — "the 7th of August" — not an
 * instant. The database column is a naive `timestamp` and the API serializes it
 * at UTC midnight (`2026-08-07T00:00:00Z`), so every time that value is handed
 * to `new Date()` it becomes an instant and gets re-read in the viewer's
 * timezone. In Argentina (UTC-3) that instant is the 6th at 21:00, so the app
 * displayed the 6th and — worse — an edit that changed nothing else submitted
 * `2026-08-06`, walking the transaction backwards one day per save.
 *
 * The fix is to convert once, explicitly, at the edges rather than sprinkling
 * offset corrections at each use:
 *
 * - coming **from** the API, read the UTC calendar parts (`calendarDateFromApi`)
 * - going **to** a date picker, build local midnight (`localDateFromCalendarDate`)
 * - coming **from** a picker, read the local calendar parts (`calendarDateFromLocalDate`)
 *
 * Between those, a date travels as a `yyyy-MM-dd` string, which is also exactly
 * what the contract's `DateString` wants on writes. No step consults the host
 * offset, so the round trip is stable in every timezone.
 */

/** A calendar date in `yyyy-MM-dd`, the contract's `DateString` form. */
export type CalendarDate = string;

const CALENDAR_DATE = /^\d{4}-\d{2}-\d{2}$/;

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

/**
 * Converts an API date into its calendar date.
 *
 * Reads **UTC** parts, because that is the timezone the API labels the value
 * with. `new Date(iso).getDate()` would instead re-interpret it locally and
 * return the previous day for any viewer west of Greenwich.
 *
 * Already-plain `yyyy-MM-dd` input is passed through, so this is safe to apply
 * to a value that has been normalized once already.
 */
export function calendarDateFromApi(value: string): CalendarDate {
  if (CALENDAR_DATE.test(value)) {
    return value;
  }

  const instant = new Date(value);
  if (Number.isNaN(instant.getTime())) {
    throw new Error(`Not a valid transaction date: ${value}`);
  }

  return `${instant.getUTCFullYear()}-${pad(instant.getUTCMonth() + 1)}-${pad(
    instant.getUTCDate()
  )}`;
}

/**
 * Builds the `Date` a date picker should show for a calendar date.
 *
 * Constructed from parts rather than parsed from the string: `new Date(
 * '2026-08-07')` is defined to parse a date-only string as **UTC** midnight,
 * which reintroduces exactly the shift this module exists to remove. Passing
 * the parts to the constructor yields local midnight on the intended day in
 * every timezone.
 */
export function localDateFromCalendarDate(value: CalendarDate): Date {
  const match = CALENDAR_DATE.exec(value);
  if (!match) {
    throw new Error(`Not a calendar date: ${value}`);
  }

  const [year, month, day] = value.split('-').map(Number);
  return new Date(year, month - 1, day);
}

/**
 * Reads the calendar date a user picked.
 *
 * Local parts, because a picker's `Date` is local midnight on the chosen day.
 * `toISOString().slice(0, 10)` would convert to UTC first and file a date
 * picked on the 7th as the 6th for anyone east of Greenwich — the same class of
 * bug in the opposite direction.
 */
export function calendarDateFromLocalDate(date: Date): CalendarDate {
  if (Number.isNaN(date.getTime())) {
    throw new Error('Not a valid date');
  }

  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

/**
 * Convenience for the read path: an API date straight to a picker `Date`.
 *
 * Exists so consumers cannot accidentally compose the two halves in the wrong
 * order, which is what produced the original off-by-one.
 */
export function localDateFromApi(value: string): Date {
  return localDateFromCalendarDate(calendarDateFromApi(value));
}
