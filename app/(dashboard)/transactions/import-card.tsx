import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  parseImportedTransactionRows,
  type ImportedTransactionRow,
  type ImportedTransactionRowError,
  type RawImportedTransactionRow,
} from '@/lib/transaction-import';
import { useState } from 'react';
import { ImportTable } from './import-table';
import type { ImportableTransactionField } from './table-head-select';

const requiredOptions = ['amount', 'date', 'payee'] as const;

type SelectedColumnsState = Record<string, ImportableTransactionField | null>;

export type { ImportedTransactionRow } from '@/lib/transaction-import';

type Props = {
  data: string[][];
  onCancel: () => void;

  onSubmit: (data: ImportedTransactionRow[]) => void;
};

export const ImportCard = ({ data, onCancel, onSubmit }: Props) => {
  const [selectedColumns, setSelectedColumns] = useState<SelectedColumnsState>(
    {}
  );
  const [importErrors, setImportErrors] = useState<
    ImportedTransactionRowError[]
  >([]);

  const headers = data[0];
  const body = data.slice(1);
  const onTableHeadSelectChange = (
    columnIndex: number,
    value: ImportableTransactionField | null
  ) => {
    setSelectedColumns((prev) => {
      const newSelectedColumns = { ...prev };
      for (const key in newSelectedColumns) {
        if (value !== null && newSelectedColumns[key] === value) {
          newSelectedColumns[key] = null;
        }
      }

      newSelectedColumns[`column_${columnIndex}`] = value;

      return newSelectedColumns;
    });
  };

  const progress = Object.values(selectedColumns).filter(
    (v) => v !== null
  ).length;

  const handleContinue = () => {
    const activeColumns = headers.reduce<
      { index: number; header: ImportableTransactionField }[]
    >((acc, _header, index) => {
      const mapped = selectedColumns[`column_${index}`];
      if (mapped) acc.push({ index, header: mapped });
      return acc;
    }, []);

    const formattedData = body
      .map((row) => {
        const obj: RawImportedTransactionRow = {};
        for (const { index, header } of activeColumns) {
          obj[header] = row[index];
        }
        return obj;
      })
      .filter((obj) => Object.keys(obj).length > 0);

    const parsedImport = parseImportedTransactionRows(formattedData);

    if (parsedImport.errors.length > 0) {
      setImportErrors(parsedImport.errors);
      return;
    }

    setImportErrors([]);

    onSubmit(parsedImport.data);
  };
  return (
    <Card className="border-none drop-shadow-sm">
      <CardHeader className="gap-y-2 lg:flex lg:flex-row lg:items-center lg:justify-between">
        <CardTitle className="line-clamp-1 text-xl">
          Import Transaction
        </CardTitle>
        <div className="flex flex-col items-center gap-x-2 gap-y-2 lg:flex-row">
          <Button onClick={onCancel} size="sm" className="w-full lg:w-auto">
            Cancel
          </Button>
          <Button
            className="w-full lg:w-auto"
            size="sm"
            disabled={progress < requiredOptions.length}
            onClick={handleContinue}
          >
            Continue ({progress} / {requiredOptions.length})
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {importErrors.length > 0 && (
          <div
            role="alert"
            className="mb-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700"
          >
            <p className="font-medium">
              Fix {importErrors.length} import value
              {importErrors.length === 1 ? '' : 's'} before continuing.
            </p>
            <ul className="mt-2 list-disc space-y-1 pl-5">
              {importErrors.slice(0, 5).map((error, index) => (
                <li key={`${error.rowNumber}-${error.field}-${index}`}>
                  Row {error.rowNumber}, {error.field}: {error.message}
                </li>
              ))}
            </ul>
            {importErrors.length > 5 && (
              <p className="mt-2">
                Showing first 5 errors. Fix those and continue to reveal any
                remaining rows.
              </p>
            )}
          </div>
        )}
        <ImportTable
          headers={headers}
          body={body}
          selectedColumns={selectedColumns}
          onTableHeadSelectChange={onTableHeadSelectChange}
        />
      </CardContent>
    </Card>
  );
};
