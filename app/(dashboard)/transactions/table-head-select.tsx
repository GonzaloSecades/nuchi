import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';

type Props = {
  columnIndex: number;
  selectedColumns: Record<string, string | null>;
  onChange: (columnIndex: number, value: string | null) => void;
};

const options = ['amount', 'date', 'payee'];

export const TableHeadSelect = ({
  columnIndex,
  selectedColumns,
  onChange,
}: Props) => {
  const currentSelection = selectedColumns[`column_${columnIndex}`];
  return (
    <Select
      value={currentSelection || ''}
      onValueChange={(value) => onChange(columnIndex, value)}
    >
      <SelectTrigger
        className={cn(
          'border-none bg-transparent capitalize outline-none focus:ring-transparent focus:ring-offset-0',
          currentSelection && 'text-blue-500',
          !currentSelection && 'text-muted-foreground italic'
        )}
      >
        <SelectValue placeholder="Skip" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="skip" className="text-muted-foreground italic">
          Skip
        </SelectItem>
        {options.map((option) => {
          const disabled =
            Object.values(selectedColumns).includes(option) &&
            selectedColumns[`column_${columnIndex}`] !== option;
          return (
            <SelectItem
              key={option}
              value={option}
              disabled={disabled}
              className="capitalize"
            >
              {option}
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
};
