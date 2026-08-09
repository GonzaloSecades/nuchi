import { describe, expect, it } from 'bun:test';
// `node:child_process` rather than `Bun.spawnSync`. `@types/bun` is present as
// of #83, so the Bun global does typecheck now, but the Node API is the more
// portable of the two and nothing here needs a Bun-specific capability.
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

/**
 * Proves the calendar-date round trip in processes that really are in a
 * different timezone.
 *
 * A process fixes its timezone at startup, so this spawns the probe under
 * several `TZ` values rather than trying to change it in-process.
 *
 * **No single spelling of a zone works on both platforms**, which is why each
 * zone carries a list of candidates rather than one string:
 *
 * - Bun on Windows resolves only the POSIX-ish `GMT±N` forms and `UTC`. IANA
 *   names are silently ignored and the process keeps the *host* zone — so
 *   `TZ=Asia/Tokyo` on a machine in Buenos Aires reports UTC−3, and a test that
 *   trusted the name would prove nothing.
 * - Bun on Linux is the mirror image: it reads the system tz database, so IANA
 *   names work and a bare `GMT-3` resolves to nothing and falls back to the
 *   host zone (UTC on a CI runner). This is not hypothetical — it is how these
 *   tests first failed in CI while passing locally.
 *
 * So a candidate is *accepted only once the spawned process reports the offset
 * the case is about*. That check was already here as a guard; it now also does
 * the selecting, which means a silent fallback can never be mistaken for a
 * working zone on either platform. If no candidate produces the offset, the
 * suite fails with everything it tried rather than testing the host zone three
 * times over.
 *
 * Note the two sign conventions in play: `GMT-3` is UTC−3 on this runtime,
 * while the IANA `Etc/GMT+3` is *also* UTC−3, because the Etc zones invert the
 * sign. They look contradictory and are both correct.
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

type Zone = {
  label: string;
  /** Spellings to try, most portable-per-platform first. */
  candidates: string[];
  /** `getTimezoneOffset()` the spawned process must report to be accepted. */
  expectedOffset: number;
};

/** West of Greenwich (Argentina), east of it (Japan), and the meridian. */
const zones: Zone[] = [
  {
    label: 'UTC-3, the app default market',
    candidates: ['GMT-3', 'Etc/GMT+3', 'America/Argentina/Buenos_Aires'],
    expectedOffset: 180,
  },
  {
    label: 'UTC+9, east of Greenwich',
    candidates: ['GMT+9', 'Etc/GMT-9', 'Asia/Tokyo'],
    expectedOffset: -540,
  },
  { label: 'UTC itself', candidates: ['UTC'], expectedOffset: 0 },
];

/**
 * Runs the probe in the first candidate spelling that actually puts the process
 * at the wanted offset.
 *
 * Memoized because the probe is a subprocess and the same zone is used by every
 * assertion in its block plus the cross-zone comparison at the end; without it
 * this file spawns the same processes nine times over.
 */
const resolved = new Map<string, ProbeResult>();

function probeZone({ label, candidates, expectedOffset }: Zone): ProbeResult {
  const cached = resolved.get(label);
  if (cached !== undefined) {
    return cached;
  }

  const observed: string[] = [];
  for (const timezone of candidates) {
    const probe = runIn(timezone);
    if (probe.offsetMinutes === expectedOffset) {
      resolved.set(label, probe);
      return probe;
    }
    observed.push(`${timezone} -> ${probe.offsetMinutes}`);
  }

  throw new Error(
    `no TZ spelling put the process at offset ${expectedOffset} for "${label}". ` +
      `Tried: ${observed.join(', ')}. Every candidate fell back to another ` +
      `zone, so these tests would have run in the host zone instead.`
  );
}

describe('transaction dates across timezones', () => {
  for (const zone of zones) {
    const { label, expectedOffset } = zone;
    describe(label, () => {
      const probe = probeZone(zone);

      // Guards the whole suite: if TZ were ignored, every case would silently
      // run in the host zone and prove nothing. probeZone already enforces
      // this while selecting; asserting it keeps the requirement visible in the
      // test output rather than buried in a helper.
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
    const probes = zones.map(probeZone);
    const naiveDays = probes.map((probe) => probe.naiveDay);
    const roundTrips = probes.map((probe) => probe.roundTrip);

    expect(naiveDays).toEqual([6, 7, 7]);
    expect(new Set(roundTrips).size).toBe(1);
    expect(roundTrips[0]).toBe('2026-08-07');
  });
});
