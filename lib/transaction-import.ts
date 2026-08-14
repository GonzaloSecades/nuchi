import { format, isValid, parse } from 'date-fns';

import { convertAmountToMiliunits, isSafeMiliunitAmount } from '@/lib/utils';

export const TRANSACTION_IMPORT_DATE_FORMAT = 'yyyy-MM-dd HH:mm:ss';
export const TRANSACTION_IMPORT_OUTPUT_DATE_FORMAT = 'yyyy-MM-dd';

const DECIMAL_AMOUNT_PATTERN = /^[+-]?(?:\d+(?:\.\d+)?|\.\d+)$/;

export type CSVParseError = {
  message: string;
  row?: number;
};

export type CSVUploadResults = {
  data: string[][];
  errors: CSVParseError[];
  meta: {
    aborted?: boolean;
    truncated?: boolean;
  };
};

export type PreparedCSVImport =
  | { ok: true; data: string[][] }
  | { ok: false; message: string };

export type ImportedTransactionRow = {
  amount: number;
  date: string;
  payee: string;
};

export type RawImportedTransactionRow = Partial<
  Record<'amount' | 'date' | 'payee', string>
>;

export type ImportedTransactionRowError = {
  rowNumber: number;
  field: 'amount' | 'date' | 'payee';
  message: string;
};

function isBlankCSVRow(row: string[]): boolean {
  return row.every((cell) => cell.trim() === '');
}

/**
 * Validates the parser result before the mapping screen can render.
 *
 * Papa Parse intentionally returns data and errors together. Treating `data`
 * as trustworthy while ignoring `errors` can turn a malformed quote or an
 * uneven record into shifted financial fields. This preflight fails closed,
 * removes only the conventional trailing blank records produced by a final
 * newline, and preserves all other row positions for useful error messages.
 */
export function prepareCSVImport(results: CSVUploadResults): PreparedCSVImport {
  if (results.meta.aborted) {
    return { ok: false, message: 'CSV parsing was interrupted.' };
  }

  if (results.meta.truncated) {
    return { ok: false, message: 'CSV parsing was truncated.' };
  }

  const firstParseError = results.errors[0];
  if (firstParseError) {
    const row =
      firstParseError.row === undefined
        ? ''
        : ` near row ${firstParseError.row + 1}`;
    return {
      ok: false,
      message: `CSV parsing failed${row}: ${firstParseError.message}`,
    };
  }

  if (
    !Array.isArray(results.data) ||
    results.data.some(
      (row) =>
        !Array.isArray(row) || row.some((cell) => typeof cell !== 'string')
    )
  ) {
    return { ok: false, message: 'CSV data has an unsupported shape.' };
  }

  const data = results.data.map((row) => [...row]);
  while (data.length > 0 && isBlankCSVRow(data[data.length - 1])) {
    data.pop();
  }

  if (data.length === 0) {
    return { ok: false, message: 'The CSV file is empty.' };
  }

  const headers = data[0].map((cell, index) =>
    (index === 0 ? cell.replace(/^\uFEFF/, '') : cell).trim()
  );

  if (headers.every((header) => header === '')) {
    return { ok: false, message: 'The CSV header row is empty.' };
  }

  if (headers.length < 3) {
    return {
      ok: false,
      message:
        'The CSV must contain at least three columns to map amount, date, and payee.',
    };
  }

  const body = data.slice(1);
  if (body.length === 0) {
    return {
      ok: false,
      message: 'The CSV contains a header but no transaction rows.',
    };
  }

  for (const [index, row] of body.entries()) {
    const rowNumber = index + 2;
    if (isBlankCSVRow(row)) {
      return { ok: false, message: `CSV row ${rowNumber} is empty.` };
    }
    if (row.length !== headers.length) {
      return {
        ok: false,
        message: `CSV row ${rowNumber} has ${row.length} columns; expected ${headers.length}.`,
      };
    }
  }

  return { ok: true, data: [headers, ...body] };
}

export function parseImportedTransactionRows(
  rows: RawImportedTransactionRow[],
  options: {
    firstRowNumber?: number;
  } = {}
) {
  const firstRowNumber = options.firstRowNumber ?? 2;
  const data: ImportedTransactionRow[] = [];
  const errors: ImportedTransactionRowError[] = [];

  rows.forEach((row, index) => {
    const rowNumber = firstRowNumber + index;
    const amountText = row.amount?.trim() ?? '';
    const dateText = row.date?.trim() ?? '';
    const payee = row.payee?.trim() ?? '';

    const amountIsNumeric =
      DECIMAL_AMOUNT_PATTERN.test(amountText) &&
      Number.isFinite(Number(amountText));
    const amount = amountIsNumeric ? Number(amountText) : Number.NaN;
    const parsedDate = parse(
      dateText,
      TRANSACTION_IMPORT_DATE_FORMAT,
      new Date()
    );

    const rowErrors: ImportedTransactionRowError[] = [];

    if (!amountIsNumeric) {
      rowErrors.push({
        rowNumber,
        field: 'amount',
        message: 'Amount must be a plain decimal number.',
      });
    }

    if (
      !dateText ||
      !isValid(parsedDate) ||
      format(parsedDate, TRANSACTION_IMPORT_DATE_FORMAT) !== dateText
    ) {
      rowErrors.push({
        rowNumber,
        field: 'date',
        message: `Date must use ${TRANSACTION_IMPORT_DATE_FORMAT}.`,
      });
    }

    if (!payee) {
      rowErrors.push({
        rowNumber,
        field: 'payee',
        message: 'Payee is required.',
      });
    }

    const amountInMiliunits = amountIsNumeric
      ? convertAmountToMiliunits(amount)
      : Number.NaN;
    if (amountIsNumeric && !isSafeMiliunitAmount(amountInMiliunits)) {
      rowErrors.push({
        rowNumber,
        field: 'amount',
        message: 'Amount is too large.',
      });
    }

    if (rowErrors.length > 0) {
      errors.push(...rowErrors);
      return;
    }

    data.push({
      amount: amountInMiliunits,
      date: format(parsedDate, TRANSACTION_IMPORT_OUTPUT_DATE_FORMAT),
      payee,
    });
  });

  return {
    data,
    errors,
  };
}
