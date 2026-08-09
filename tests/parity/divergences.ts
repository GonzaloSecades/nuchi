/**
 * The Hono → Go divergences, as data the harness actually consults.
 *
 * The harness exists to find *unintended* differences between the two stacks.
 * That only works if the intended ones are written down: without this list a
 * run is a wall of red that someone has to re-triage by hand every time, and
 * the one real regression hides among the fifteen decisions we already made on
 * purpose.
 *
 * Each entry points at where it is argued — a numbered file in
 * `post-migration-improvements/claude-backend-improvements/`, or a
 * "Divergences" section in `docs/api/`. A difference the harness observes whose
 * key is not registered here classifies as `unknown`, which is a finding.
 */

export type DivergenceKey = string;

export type Divergence = {
  /** Stable identifier the tests classify against. */
  key: DivergenceKey;
  /** Registry entry number, or null for contract-driven decisions with no entry. */
  entry: number | null;
  /** Which response surface it shows up on. */
  surface: 'transactions' | 'summary' | 'accounts' | 'categories' | 'all';
  /** What actually differs, in terms of observable behavior. */
  description: string;
  /** Whether a difference here is a decision rather than a defect. */
  expected: boolean;
};

/**
 * The key for the naive-timestamp difference this harness found.
 *
 * Exported as a constant so the test that observes it and the entry that
 * classifies it cannot drift apart through a typo.
 */
export const DATE_SERIALIZATION_TIMEZONE = 'transactions.date.timezone';

export const KNOWN_DIVERGENCES: Divergence[] = [
  {
    key: 'transactions.amount.bigint',
    entry: 6,
    surface: 'transactions',
    description:
      '`amount` widened to bigint; the legacy ~2.15M ARS cap is raised.',
    expected: true,
  },
  {
    key: 'filters.date.utc',
    entry: 14,
    surface: 'all',
    description:
      'Date range filters are parsed in UTC rather than the host timezone, so a boundary day can fall on the other side of the range.',
    expected: true,
  },
  {
    key: 'transactions.amount.out-of-range-status',
    entry: 15,
    surface: 'transactions',
    description:
      'An out-of-range amount returns 400 rather than the legacy 500.',
    expected: true,
  },
  {
    key: 'transactions.bulk.byte-limit',
    entry: 16,
    surface: 'transactions',
    description:
      'Bulk byte limits are enforced against the stream, not only Content-Length.',
    expected: true,
  },
  {
    key: 'transactions.currency.required',
    entry: null,
    surface: 'transactions',
    description:
      '`currency` is required and accepted only as ARS; the legacy schema has no such column.',
    expected: true,
  },
  {
    key: 'query.empty-parameters-rejected',
    entry: null,
    surface: 'all',
    description:
      'Explicitly empty query parameters are rejected with 400 INVALID_QUERY rather than treated as absent.',
    expected: true,
  },
  {
    key: 'transactions.order.id-tiebreak',
    entry: null,
    surface: 'transactions',
    description: '`id` is used as an ordering tiebreak after `date`.',
    expected: true,
  },

  /**
   * Found by this harness on 2026-08-09, and deliberately marked NOT expected.
   *
   * `transactions.date` is `timestamp without time zone`. node-postgres reads a
   * naive timestamp as local time; Go reads the same bytes as UTC. For a row
   * stored as `2026-08-07 00:00:00` the two stacks return
   * `2026-08-07T03:00:00.000Z` and `2026-08-07T00:00:00Z`, and a browser west
   * of Greenwich renders those as different calendar days — every transaction
   * date shows a day early, including for Argentina.
   *
   * Left `expected: false` on purpose: it is a parity regression rather than a
   * decision, and the test asserts this classification, so marking it expected
   * fails the suite rather than quietly retiring the only check that reports it.
   */
  {
    key: DATE_SERIALIZATION_TIMEZONE,
    entry: null,
    surface: 'transactions',
    description:
      'A naive `timestamp` is serialized as UTC by Go and as host-local by node-postgres, shifting the rendered calendar day west of Greenwich.',
    expected: false,
  },
];

/** How the harness treats a difference it has observed. */
export type Classification = 'expected' | 'open' | 'unknown';

/**
 * Classifies an observed difference.
 *
 * `unknown` is the important one: a difference the harness can see but nobody
 * has written down is precisely the regression this exists to catch, and it
 * must not be quietly grouped with the decisions.
 */
export function classifyDivergence(key: DivergenceKey): Classification {
  const divergence = KNOWN_DIVERGENCES.find((entry) => entry.key === key);
  if (divergence === undefined) {
    return 'unknown';
  }
  return divergence.expected ? 'expected' : 'open';
}

/** Looks up a registered divergence, or undefined when it is not registered. */
export function findDivergence(key: DivergenceKey): Divergence | undefined {
  return KNOWN_DIVERGENCES.find((entry) => entry.key === key);
}

/** The divergences the harness should not treat as failures. */
export const EXPECTED_DIVERGENCES = KNOWN_DIVERGENCES.filter(
  (divergence) => divergence.expected
);

/** Recorded differences still considered defects. */
export const OPEN_DIVERGENCES = KNOWN_DIVERGENCES.filter(
  (divergence) => !divergence.expected
);
