import { describe, expect, it } from 'bun:test';

import {
  parseImportedTransactionRows,
  prepareCSVImport,
  type CSVUploadResults,
} from '@/lib/transaction-import';

function uploadResult(
  data: string[][],
  options: Partial<Pick<CSVUploadResults, 'errors' | 'meta'>> = {}
): CSVUploadResults {
  return {
    data,
    errors: options.errors ?? [],
    meta: options.meta ?? {},
  };
}

describe('prepareCSVImport', () => {
  it('normalizes headers and removes only trailing blank records', () => {
    expect(
      prepareCSVImport(
        uploadResult([
          ['\uFEFF Amount ', ' Date ', ' Payee '],
          ['12.34', '2026-04-17 13:45:00', 'Coffee Shop'],
          ['', '', ''],
        ])
      )
    ).toEqual({
      ok: true,
      data: [
        ['Amount', 'Date', 'Payee'],
        ['12.34', '2026-04-17 13:45:00', 'Coffee Shop'],
      ],
    });
  });

  it('rejects an empty file and an empty header', () => {
    expect(prepareCSVImport(uploadResult([]))).toEqual({
      ok: false,
      message: 'The CSV file is empty.',
    });
    expect(
      prepareCSVImport(
        uploadResult([
          ['', '  ', ''],
          ['1', '2', '3'],
        ])
      )
    ).toEqual({
      ok: false,
      message: 'The CSV header row is empty.',
    });
  });

  it('rejects a header-only file before the mapping screen', () => {
    expect(
      prepareCSVImport(uploadResult([['Amount', 'Date', 'Payee']]))
    ).toEqual({
      ok: false,
      message: 'The CSV contains a header but no transaction rows.',
    });
  });

  it('rejects parser errors instead of trusting partial parser data', () => {
    expect(
      prepareCSVImport(
        uploadResult(
          [
            ['Amount', 'Date', 'Payee'],
            ['12.34', '2026-04-17 13:45:00', 'Coffee Shop'],
          ],
          { errors: [{ row: 1, message: 'Quoted field unterminated' }] }
        )
      )
    ).toEqual({
      ok: false,
      message: 'CSV parsing failed near row 2: Quoted field unterminated',
    });
  });

  it('rejects aborted and truncated parser results', () => {
    const data = [
      ['Amount', 'Date', 'Payee'],
      ['12.34', '2026-04-17 13:45:00', 'Coffee Shop'],
    ];

    expect(
      prepareCSVImport(uploadResult(data, { meta: { aborted: true } }))
    ).toEqual({ ok: false, message: 'CSV parsing was interrupted.' });
    expect(
      prepareCSVImport(uploadResult(data, { meta: { truncated: true } }))
    ).toEqual({ ok: false, message: 'CSV parsing was truncated.' });
  });

  it('requires enough columns for every required mapping', () => {
    expect(
      prepareCSVImport(
        uploadResult([
          ['Amount', 'Date'],
          ['12.34', '2026-04-17 13:45:00'],
        ])
      )
    ).toEqual({
      ok: false,
      message:
        'The CSV must contain at least three columns to map amount, date, and payee.',
    });
  });

  it('fails closed on blank or uneven records inside the file', () => {
    const header = ['Amount', 'Date', 'Payee'];

    expect(
      prepareCSVImport(
        uploadResult([
          header,
          ['12.34', '2026-04-17 13:45:00', 'Coffee Shop'],
          ['', '', ''],
          ['9.50', '2026-04-18 13:45:00', 'Market'],
        ])
      )
    ).toEqual({ ok: false, message: 'CSV row 3 is empty.' });

    expect(
      prepareCSVImport(uploadResult([header, ['12.34', '2026-04-17 13:45:00']]))
    ).toEqual({
      ok: false,
      message: 'CSV row 2 has 2 columns; expected 3.',
    });
  });
});

describe('parseImportedTransactionRows', () => {
  it('converts valid imported rows to API transaction rows', () => {
    const result = parseImportedTransactionRows([
      {
        amount: ' 12.34 ',
        date: ' 2026-04-17 13:45:00 ',
        payee: ' Coffee Shop ',
      },
    ]);

    expect(result.errors).toEqual([]);
    expect(result.data).toEqual([
      {
        amount: 12340,
        date: '2026-04-17',
        payee: 'Coffee Shop',
      },
    ]);
  });

  it('returns one useful field error for each malformed value', () => {
    const result = parseImportedTransactionRows([
      {
        amount: 'not-a-number',
        date: 'not-a-date',
        payee: '',
      },
    ]);

    expect(result.data).toEqual([]);
    expect(result.errors).toHaveLength(3);
    expect(result.errors.map((error) => error.field)).toEqual([
      'amount',
      'date',
      'payee',
    ]);
  });

  it('rejects hexadecimal and exponent notation instead of coercing it', () => {
    const result = parseImportedTransactionRows([
      {
        amount: '0x10',
        date: '2026-04-17 13:45:00',
        payee: 'Hex',
      },
      {
        amount: '1e3',
        date: '2026-04-18 13:45:00',
        payee: 'Exponent',
      },
    ]);

    expect(result.data).toEqual([]);
    expect(result.errors).toEqual([
      {
        rowNumber: 2,
        field: 'amount',
        message: 'Amount must be a plain decimal number.',
      },
      {
        rowNumber: 3,
        field: 'amount',
        message: 'Amount must be a plain decimal number.',
      },
    ]);
  });

  it('reports an out-of-range amount without a duplicate syntax error', () => {
    const result = parseImportedTransactionRows([
      {
        amount: '9007199254741',
        date: '2026-04-17 13:45:00',
        payee: 'Large transfer',
      },
    ]);

    expect(result.data).toEqual([]);
    expect(result.errors).toEqual([
      {
        rowNumber: 2,
        field: 'amount',
        message: 'Amount is too large.',
      },
    ]);
  });

  it('supports an explicit physical first-row number', () => {
    const result = parseImportedTransactionRows(
      [{ amount: '', date: '', payee: '' }],
      { firstRowNumber: 8 }
    );

    expect(result.errors.every((error) => error.rowNumber === 8)).toBe(true);
  });
});
