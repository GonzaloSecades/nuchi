import { format, isValid, parse } from 'date-fns';

import { convertAmountToMiliunits } from '@/lib/utils';

export const TRANSACTION_IMPORT_DATE_FORMAT = 'yyyy-MM-dd HH:mm:ss';
export const TRANSACTION_IMPORT_OUTPUT_DATE_FORMAT = 'yyyy-MM-dd';

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

    const amount = Number(amountText);
    const parsedDate = parse(
      dateText,
      TRANSACTION_IMPORT_DATE_FORMAT,
      new Date()
    );

    const rowErrors: ImportedTransactionRowError[] = [];

    if (!amountText || !Number.isFinite(amount)) {
      rowErrors.push({
        rowNumber,
        field: 'amount',
        message: 'Amount must be a valid number.',
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

    if (rowErrors.length > 0) {
      errors.push(...rowErrors);
      return;
    }

    data.push({
      amount: convertAmountToMiliunits(amount),
      date: format(parsedDate, TRANSACTION_IMPORT_OUTPUT_DATE_FORMAT),
      payee,
    });
  });

  return {
    data,
    errors,
  };
}
