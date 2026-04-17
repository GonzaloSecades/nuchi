import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { convertAmountToMiliunits } from '@/lib/utils';
import { format, parse } from 'date-fns';
import { useState } from 'react';
import { ImportTable } from './import-table';
import type { ImportableTransactionField } from './table-head-select';

const dateFormat = 'yyyy-MM-dd HH:mm:ss';
const outputFormat = 'yyyy-MM-dd';

const requiredOptions = ['amount', 'date', 'payee'] as const;

type SelectedColumnsState = Record<string, ImportableTransactionField | null>;

type RawImportRow = Partial<Record<ImportableTransactionField, string>>;

export type ImportedTransactionRow = {
  amount: number;
  date: string;
  payee: string;
};

type Props = {
  data: string[][];
  onCancel: () => void;

  onSubmit: (data: ImportedTransactionRow[]) => void;
};

export const ImportCard = ({ data, onCancel, onSubmit }: Props) => {
  const [selectedColumns, setSelectedColumns] = useState<SelectedColumnsState>(
    {}
  );

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
        const obj: RawImportRow = {};
        for (const { index, header } of activeColumns) {
          obj[header] = row[index];
        }
        return obj;
      })
      .filter((obj) => Object.keys(obj).length > 0)
      .map((item) => ({
        amount: convertAmountToMiliunits(parseFloat(item.amount ?? '0')),
        date: format(
          parse(item.date ?? '', dateFormat, new Date()),
          outputFormat
        ),
        payee: item.payee ?? '',
      }));

    onSubmit(formattedData);
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
