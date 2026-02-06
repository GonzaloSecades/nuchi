import { format } from 'date-fns';
import { Calendar as CalendarIcon } from 'lucide-react';

import { OnSelectHandler } from 'react-day-picker';

import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { cn } from '@/lib/utils';

type Props = {
  value?: Date;
  onChange: OnSelectHandler<Date | undefined>;
  disabled?: boolean;
};

export const DatePicker = ({ value, onChange, disabled }: Props) => {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          disabled={disabled}
          variant="outline"
          className={cn(
            'w-full justify-start text-left font-normal',
            !value && 'text-muted-foreground'
          )}
        >
          <CalendarIcon className="mr-2 size-4" />
          {value ? format(value, 'PPP') : <span>Pick a date</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent>
        <Calendar
          mode="single"
          selected={value}
          onSelect={onChange}
          disabled={disabled}
          autoFocus
        />
      </PopoverContent>
    </Popover>
  );
};
