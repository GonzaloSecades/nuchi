import { describe, expect, it } from 'bun:test';

import {
  classifyDivergence,
  DATE_SERIALIZATION_TIMEZONE,
  EXPECTED_DIVERGENCES,
  findDivergence,
  KNOWN_DIVERGENCES,
  OPEN_DIVERGENCES,
} from './divergences';

/**
 * The registry is only useful if the harness consults it, and only trustworthy
 * if the classification itself is tested. These run with no database and no Go
 * API, so the rules stay verified even when a full parity run is skipped.
 */

describe('classifyDivergence', () => {
  it('classifies a deliberate decision as expected', () => {
    // Entry 0014: date filters parsed in UTC.
    expect(classifyDivergence('filters.date.utc')).toBe('expected');
  });

  it('classifies a recorded defect as open', () => {
    expect(classifyDivergence(DATE_SERIALIZATION_TIMEZONE)).toBe('open');
  });

  /**
   * The case that matters most: a difference nobody wrote down is a finding,
   * not something to quietly file with the decisions.
   */
  it('classifies an unregistered difference as unknown', () => {
    expect(classifyDivergence('summary.rounding.half-up')).toBe('unknown');
    expect(classifyDivergence('')).toBe('unknown');
  });
});

describe('the registry itself', () => {
  it('has a unique key for every entry', () => {
    const keys = KNOWN_DIVERGENCES.map((entry) => entry.key);
    expect(new Set(keys).size).toBe(keys.length);
  });

  it('gives every entry a non-empty key and description', () => {
    for (const entry of KNOWN_DIVERGENCES) {
      expect(entry.key.length).toBeGreaterThan(0);
      expect(entry.description.length).toBeGreaterThan(0);
    }
  });

  it('splits cleanly into expected and open', () => {
    expect(EXPECTED_DIVERGENCES.length + OPEN_DIVERGENCES.length).toBe(
      KNOWN_DIVERGENCES.length
    );
    expect(EXPECTED_DIVERGENCES.every((entry) => entry.expected)).toBe(true);
    expect(OPEN_DIVERGENCES.every((entry) => !entry.expected)).toBe(true);
  });

  it('finds a registered entry and reports nothing for an unregistered one', () => {
    expect(findDivergence(DATE_SERIALIZATION_TIMEZONE)?.surface).toBe(
      'transactions'
    );
    expect(findDivergence('nope')).toBeUndefined();
  });

  /**
   * Guards the intent stated in the README: the date divergence is a
   * regression, not a decision. Marking it expected would suppress the only
   * check that reports it, so that change has to fail here first.
   */
  it('keeps the date divergence recorded as an open defect', () => {
    const entry = findDivergence(DATE_SERIALIZATION_TIMEZONE);

    expect(entry).toBeDefined();
    expect(entry?.expected).toBe(false);
  });
});
