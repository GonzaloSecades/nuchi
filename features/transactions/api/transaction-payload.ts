import type { components } from '@/lib/api/generated/schema';
import {
  calendarDateFromLocalDate,
  type CalendarDate,
} from '@/features/transactions/api/transaction-date';

/** The contract's write shape for a transaction. */
export type TransactionInput = components['schemas']['TransactionInput'];

/**
 * What the transaction form hands a mutation hook.
 *
 * Derived from the contract while preserving the UI boundary: the form holds
 * a local `Date`, and optional nullable fields may be omitted. The adapter
 * below supplies the wire-format calendar date and required currency.
 */
export type TransactionFormValues = Omit<
  TransactionInput,
  'date' | 'currency' | 'categoryId' | 'notes'
> & {
  date: Date;
  categoryId?: TransactionInput['categoryId'];
  notes?: TransactionInput['notes'];
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
 * is `yyyy-MM-dd`; the form holds a `Date`. The conversion goes through
 * `transaction-date.ts`, which reads *local* calendar parts.
 * `toISOString().slice(0, 10)` would convert to UTC first and hand back the
 * previous day for any user east of Greenwich — a transaction picked as the 9th
 * filed as the 8th, silently, and only for some users.
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
    date: calendarDateFromLocalDate(values.date),
    accountId: values.accountId,
    // `''` normalizes to null alongside undefined. The contract's `ResourceId`
    // sets `minLength: 1`, so an empty string is rejected outright — and an
    // empty string is exactly what an unselected `<select>` or a cleared form
    // field produces. Absorbing it here keeps that from reaching the API as a
    // guaranteed 400.
    categoryId: emptyToNull(values.categoryId),
    currency: DEFAULT_CURRENCY,
  };
}

/** Treats an unset, null or blank optional id as absent. */
function emptyToNull(value: string | null | undefined): string | null {
  return value === undefined || value === null || value === '' ? null : value;
}

/**
 * One row submitted to bulk create.
 *
 * `date` is a `yyyy-MM-dd` string rather than a `Date` because the CSV parser
 * (`parseImportedTransactionRows`) already formats it that way, so that path
 * needs no date conversion.
 *
 * `notes` and `categoryId` are optional but **not** ignored. The legacy hook
 * took the full transaction insert shape, so narrowing the type and hardcoding
 * both to `null` would silently discard whatever a caller passed — a payload
 * accepted and then quietly altered, which is worse than a type error. The CSV
 * importer supplies neither today; the contract still requires both keys to be
 * present, so omitted values normalize to `null`.
 */
export type ImportedTransactionInput = {
  amount: number;
  date: CalendarDate;
  payee: string;
  accountId: string;
  notes?: string | null;
  categoryId?: string | null;
};

/** Converts bulk-create rows into the contract's bare-array body. */
export function toBulkTransactionInput(
  rows: ImportedTransactionInput[]
): TransactionInput[] {
  return rows.map((row) => ({
    amount: row.amount,
    payee: row.payee,
    notes: row.notes ?? null,
    categoryId: emptyToNull(row.categoryId),
    date: row.date,
    accountId: row.accountId,
    currency: DEFAULT_CURRENCY,
  }));
}
