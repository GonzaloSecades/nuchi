import { format } from 'date-fns';

import type { components } from '@/lib/api/generated/schema';

/** The contract's write shape for a transaction. */
export type TransactionInput = components['schemas']['TransactionInput'];

/**
 * What the transaction form hands a mutation hook.
 *
 * Derived from the Drizzle insert schema rather than the contract, because
 * that is what the form and both sheets already produce. Keeping the hooks'
 * public input at this shape is what lets #50 migrate the transport without
 * touching a single component.
 */
export type TransactionFormValues = {
  date: Date;
  accountId: string;
  categoryId?: string | null;
  payee: string;
  /** Signed integer milliunits, already converted by the form. */
  amount: number;
  notes?: string | null;
};

/**
 * The currency every write carries.
 *
 * The contract requires `currency` and accepts only `ARS`, and deliberately
 * rejects an omitted value rather than defaulting it — so a client that
 * believes it is sending something else fails loudly instead of silently
 * booking pesos. The legacy Hono route has no currency at all, so nothing
 * upstream of here supplies one; this is the single place the app states it,
 * and the place to change when multi-currency arrives.
 */
const DEFAULT_CURRENCY = 'ARS' as const;

/**
 * Converts form values into the contract's `TransactionInput`.
 *
 * Two things the form does not know about, and the reason this adapter exists
 * rather than the components being changed:
 *
 * **`date` is a calendar date, not an instant.** The contract's `DateString`
 * is `yyyy-MM-dd`; the form holds a `Date`. Formatting goes through date-fns,
 * which reads *local* calendar parts. `toISOString().slice(0, 10)` would
 * convert to UTC first and hand back the previous day for any user east of
 * Greenwich — a transaction picked as the 9th filed as the 8th, silently, and
 * only for some users.
 *
 * **`currency` is required.** See `DEFAULT_CURRENCY`.
 */
export function toTransactionInput(
  values: TransactionFormValues
): TransactionInput {
  return {
    amount: values.amount,
    payee: values.payee,
    notes: values.notes ?? null,
    date: format(values.date, 'yyyy-MM-dd'),
    accountId: values.accountId,
    categoryId: values.categoryId ?? null,
    currency: DEFAULT_CURRENCY,
  };
}

/**
 * One CSV row, as the import screen produces it.
 *
 * `date` is already a `yyyy-MM-dd` string here rather than a `Date` —
 * `parseImportedTransactionRows` formats it on the way out — so this shape
 * needs no date conversion, only the currency the contract requires.
 */
export type ImportedTransactionInput = {
  amount: number;
  date: string;
  payee: string;
  accountId: string;
};

/** Converts imported CSV rows into the contract's bulk-create body. */
export function toBulkTransactionInput(
  rows: ImportedTransactionInput[]
): TransactionInput[] {
  return rows.map((row) => ({
    amount: row.amount,
    payee: row.payee,
    // A CSV import carries neither, and the generated type requires both to be
    // present even though the contract gives them a null default.
    notes: null,
    categoryId: null,
    date: row.date,
    accountId: row.accountId,
    currency: DEFAULT_CURRENCY,
  }));
}
