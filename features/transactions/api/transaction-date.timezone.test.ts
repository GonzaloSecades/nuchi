import { describe, expect, it } from 'bun:test';
// `node:child_process` rather than `Bun.spawnSync`: the repo has `@types/node`
// but not `@types/bun`, so the Bun global is untyped and would fail `tsc`.
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

/**
 * Proves the calendar-date round trip in processes that really are in a
 * different timezone.
 *
 * A process fixes its timezone at startup, so this spawns the probe under
 * several `TZ` values rather than trying to change it in-process. Offsets of
 * the form `GMT+9` are used instead of IANA names (`Asia/Tokyo`) because Bun on
 * Windows silently ignores IANA names and falls back to the host zone — which
 * would make these tests pass without testing anything.
 *
 * On this runtime `GMT+9` resolves to UTC+9 and `GMT-3` to UTC-3, i.e. the sign
 * reads the intuitive way. That is worth stating because strict POSIX defines
 * it the other way round, where `GMT+9` would mean UTC-9. Each case asserts the
 * offset it actually observed before asserting anything else, so a runtime that
 * followed the POSIX reading would fail loudly here rather than quietly test a
 * zone nobody intended.
 */

// `fileURLToPath` rather than `URL.pathname`: on Windows the latter yields a
// leading-slash path like `/C:/dev/...` that Bun cannot resolve.
const PROBE = fileURLToPath(
  new URL('./transaction-date.timezone-probe.ts', import.meta.url)
);
const API_INSTANT = '2026-08-07T00:00:00Z';

type ProbeResult = {
  offsetMinutes: number;
  calendarDate: string;
  pickerDay: number;
  pickerMonth: number;
  roundTrip: string;
  naiveDay: number;
};

function runIn(timezone: string): ProbeResult {
  const result = spawnSync('bun', ['run', PROBE, API_INSTANT], {
    env: { ...process.env, TZ: timezone },
    encoding: 'utf8',
    shell: process.platform === 'win32',
  });

  if (result.status !== 0) {
    throw new Error(
      `probe failed in ${timezone}: ${String(result.stderr).slice(0, 400)}`
    );
  }

  return JSON.parse(result.stdout) as ProbeResult;
}

/** West of Greenwich (Argentina), east of it (Japan), and the meridian. */
const zones: Array<[label: string, tz: string, expectedOffset: number]> = [
  ['UTC-3, the app default market', 'GMT-3', 180],
  ['UTC+9, east of Greenwich', 'GMT+9', -540],
  ['UTC itself', 'UTC', 0],
];

describe('transaction dates across timezones', () => {
  for (const [label, timezone, expectedOffset] of zones) {
    describe(label, () => {
      const probe = runIn(timezone);

      // Guards the whole suite: if TZ were ignored, every case would silently
      // run in the host zone and prove nothing.
      it('really runs in that timezone', () => {
        expect(probe.offsetMinutes).toBe(expectedOffset);
      });

      it('reads the stored calendar date', () => {
        expect(probe.calendarDate).toBe('2026-08-07');
      });

      it('shows the stored day in the edit picker', () => {
        expect(probe.pickerDay).toBe(7);
        expect(probe.pickerMonth).toBe(8);
      });

      it('submits the same date when nothing was changed', () => {
        expect(probe.roundTrip).toBe('2026-08-07');
      });
    });
  }

  /**
   * The bug this replaced, demonstrated rather than described: reading the API
   * value as an instant lands on the 6th west of Greenwich and the 7th
   * elsewhere. The conversion above is the same in all three.
   */
  it('is stable where the previous instant-based reading was not', () => {
    const naiveDays = zones.map(([, tz]) => runIn(tz).naiveDay);
    const roundTrips = zones.map(([, tz]) => runIn(tz).roundTrip);

    expect(naiveDays).toEqual([6, 7, 7]);
    expect(new Set(roundTrips).size).toBe(1);
    expect(roundTrips[0]).toBe('2026-08-07');
  });
});
