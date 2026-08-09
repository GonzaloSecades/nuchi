/**
 * Runs the transaction date round trip and reports the result as JSON.
 *
 * Spawned by `transaction-date.timezone.test.ts` under different `TZ` values.
 * It exists as a separate entry point because a process's timezone is fixed at
 * startup — the only way to prove the conversion holds east and west of
 * Greenwich is to run it in a process that genuinely is there.
 *
 * Not named `*.test.ts` so the test runner does not collect it directly.
 */

import {
  calendarDateFromApi,
  calendarDateFromLocalDate,
  localDateFromApi,
} from './transaction-date';

const apiInstant = process.argv[2] ?? '2026-08-07T00:00:00Z';

const picker = localDateFromApi(apiInstant);

process.stdout.write(
  JSON.stringify({
    offsetMinutes: new Date().getTimezoneOffset(),
    calendarDate: calendarDateFromApi(apiInstant),
    // What the edit sheet shows.
    pickerDay: picker.getDate(),
    pickerMonth: picker.getMonth() + 1,
    // What an unchanged save submits.
    roundTrip: calendarDateFromLocalDate(picker),
    // The old behaviour, kept for contrast: reading the API value as an instant.
    naiveDay: new Date(apiInstant).getDate(),
  })
);
