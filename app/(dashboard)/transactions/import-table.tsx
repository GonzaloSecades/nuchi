import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { TableHeadSelect } from './table-head-select';

type Props = {
  headers: string[];
  body: string[][];
  selectedColumns: Record<string, string | null>;
  onTableHeadSelectChange: (columnIndex: number, value: string | null) => void;
};

export const ImportTable = ({
  headers,
  body,
  selectedColumns,
  onTableHeadSelectChange,
}: Props) => {
  const rowKeyCounts = new Map<string, number>();

  return (
    <div className="overflow-hidden rounded-md border">
      <Table>
        <TableHeader className="bg-muted">
          <TableRow>
            {headers.map((header, index) => (
              <TableHead key={header || `column-${index}`}>
                <TableHeadSelect
                  columnIndex={index}
                  selectedColumns={selectedColumns}
                  onChange={onTableHeadSelectChange}
                />
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {body.map((row: string[]) => {
            const rowKeyBase = row.join('|');
            const rowKeyCount = rowKeyCounts.get(rowKeyBase) ?? 0;
            rowKeyCounts.set(rowKeyBase, rowKeyCount + 1);
            const rowKey = rowKeyCount
              ? `${rowKeyBase}-${rowKeyCount}`
              : rowKeyBase;

            return (
              <TableRow key={rowKey}>
                {row.map((cell, cellIndex) => {
                  const headerKey = headers[cellIndex] || `column-${cellIndex}`;
                  return (
                    <TableCell key={`${rowKey}-${headerKey}`}>{cell}</TableCell>
                  );
                })}
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
};

/**
 * Keys rationale:
 * - Header keys use the header label when available, falling back to a stable `column-${index}`.
 * - Row keys are derived from the row content to remain stable across re-renders.
 * - If the same row content appears more than once, a numeric suffix is appended to keep keys unique.
 *
 * Transaction-identity note:
 * - A real duplicate transaction is usually identified by fields like date, amount, payee, and account.
 *
 * Future improvement:
 * - If the import provides a unique transaction ID (or a canonical business key), prefer that as the row key.
 */
