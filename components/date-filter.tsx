'use client';

import {
  endOfMonth,
  endOfWeek,
  endOfYear,
  format,
  isValid,
  parse,
  startOfMonth,
  startOfWeek,
  startOfYear,
  subDays,
  subMonths,
  subWeeks,
  subYears,
} from 'date-fns';
import { ChevronDown, RotateCcw } from 'lucide-react';
import { useState } from 'react';
import { DateRange } from 'react-day-picker';

import qs from 'query-string';

import { usePathname, useRouter, useSearchParams } from 'next/navigation';

import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import {
  Popover,
  PopoverClose,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { formatDateRange } from '@/lib/utils';

type PresetKey =
  | 'today'
  | 'yesterday'
  | 'thisWeek'
  | 'lastWeek'
  | 'thisMonth'
  | 'lastMonth'
  | 'thisYear'
  | 'lastYear'
  | 'allTime';

type Preset = {
  key: PresetKey;
  label: string;
};

const PRESETS: Preset[] = [
  { key: 'today', label: 'Today' },
  { key: 'yesterday', label: 'Yesterday' },
  { key: 'thisWeek', label: 'This week' },
  { key: 'lastWeek', label: 'Last week' },
  { key: 'thisMonth', label: 'This month' },
  { key: 'lastMonth', label: 'Last month' },
  { key: 'thisYear', label: 'This year' },
  { key: 'lastYear', label: 'Last year' },
  { key: 'allTime', label: 'All time' },
];

export const DateFilter = () => {
  const router = useRouter();
  const pathname = usePathname();

  const params = useSearchParams();
  const accountId = params.get('accountId');
  const from = params.get('from') || '';
  const to = params.get('to') || '';

  const defaultTo = new Date();
  const defaultFrom = subDays(defaultTo, 30);

  const parseDateParam = (value: string, fallback: Date) => {
    if (!value) return fallback;
    const parsed = parse(value, 'yyyy-MM-dd', new Date());
    return isValid(parsed) ? parsed : fallback;
  };

  const paramState = {
    from: parseDateParam(from, defaultFrom),
    to: parseDateParam(to, defaultTo),
  };

  const [open, setOpen] = useState(false);
  const [date, setDate] = useState<DateRange | undefined>(paramState);
  const [month, setMonth] = useState<Date>(paramState.from);

  const onOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);

    if (nextOpen) {
      // re-seed from URL-derived state every time the popover opens
      setDate(paramState);
      setMonth(paramState.from);
    }
  };

  const pushToUrl = (dateRange: DateRange | undefined) => {
    const query = {
      from: format(dateRange?.from || defaultFrom, 'yyyy-MM-dd'),
      to: format(dateRange?.to || defaultTo, 'yyyy-MM-dd'),
      accountId,
    };

    const url = qs.stringifyUrl(
      {
        url: pathname,
        query,
      },
      { skipEmptyString: true, skipNull: true }
    );

    router.push(url);
  };

  const onReset = () => {
    setDate(undefined);
    pushToUrl(undefined);
  };

  const onPresetSelect = (preset: PresetKey) => {
    const now = new Date();
    let next: DateRange;

    switch (preset) {
      case 'today': {
        next = { from: now, to: now };
        break;
      }
      case 'yesterday': {
        const y = subDays(now, 1);
        next = { from: y, to: y };
        break;
      }
      case 'thisWeek': {
        next = {
          from: startOfWeek(now, { weekStartsOn: 1 }),
          to: endOfWeek(now, { weekStartsOn: 1 }),
        };
        break;
      }
      case 'lastWeek': {
        const lastWeekDate = subWeeks(now, 1);
        next = {
          from: startOfWeek(lastWeekDate, { weekStartsOn: 1 }),
          to: endOfWeek(lastWeekDate, { weekStartsOn: 1 }),
        };
        break;
      }
      case 'thisMonth': {
        next = { from: startOfMonth(now), to: endOfMonth(now) };
        break;
      }
      case 'lastMonth': {
        const lastMonthDate = subMonths(now, 1);
        next = {
          from: startOfMonth(lastMonthDate),
          to: endOfMonth(lastMonthDate),
        };
        break;
      }
      case 'thisYear': {
        next = { from: startOfYear(now), to: endOfYear(now) };
        break;
      }
      case 'lastYear': {
        const lastYearDate = subYears(now, 1);
        next = {
          from: startOfYear(lastYearDate),
          to: endOfYear(lastYearDate),
        };
        break;
      }
      case 'allTime': {
        // Adjust to your product's "true min date" if you have one in DB
        next = { from: new Date(2000, 0, 1), to: now };
        break;
      }
      default: {
        next = { from: defaultFrom, to: defaultTo };
      }
    }

    setDate(next);

    if (preset !== 'allTime' && next.from) {
      setMonth(next.from);
    }
  };

  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger asChild>
        <Button
          disabled={false}
          size="sm"
          variant="outline"
          className="h-9 w-full rounded-md border-none bg-white/10 px-3 font-normal text-white transition outline-none hover:bg-white/20 hover:text-white focus:bg-white/30 focus:ring-transparent focus:ring-offset-0 lg:w-auto"
        >
          <span>{formatDateRange(paramState)}</span>
          <ChevronDown className="ml-2 size-4 opacity-50" />
        </Button>
      </PopoverTrigger>
      {/* <PopoverContent className="w-full p-0 lg:w-auto" align="start">
        <div className="flex items-center gap-2 border-b p-4">
          <Calendar
            disabled={false}
            autoFocus={true}
            mode="range"
            defaultMonth={date?.from}
            selected={date}
            onSelect={setDate}
            numberOfMonths={2}
          />
          <div>Hola</div>
        </div>
        <div className="flex w-full items-center gap-2 p-4">
          <PopoverClose asChild>
            <Button
              onClick={onReset}
              disabled={!date?.from || !date?.to}
              className="h-9 w-9 p-0"
              variant="outline"
            >
              <RotateCcw />
            </Button>
          </PopoverClose>
          <PopoverClose asChild>
            <Button
              onClick={() => {
                pushToUrl(date);
              }}
              disabled={!date?.from || !date?.to}
              className="h-9 flex-1 basis-1/2"
            >
              Apply
            </Button>
          </PopoverClose>
        </div>
      </PopoverContent> */}
      <PopoverContent
        align="start"
        className="w-auto max-w-[calc(100vw-2rem)] overflow-hidden rounded-xl p-0"
      >
        <div className="flex w-fit">
          {/* Presets column */}
          <aside className="bg-muted/20 w-42.5 shrink-0 border-r p-3">
            <div className="flex flex-col items-start gap-1">
              {PRESETS.map((preset) => (
                <Button
                  key={preset.key}
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => onPresetSelect(preset.key)}
                  className="hover:bg-muted h-8 w-fit justify-start px-2 text-sm font-normal"
                >
                  {preset.label}
                </Button>
              ))}
            </div>
          </aside>

          {/* Calendar + actions */}
          <div className="flex-1">
            <div className="w-fit border-b p-3">
              <Calendar
                disabled={false}
                autoFocus
                mode="range"
                month={month}
                onMonthChange={setMonth}
                defaultMonth={date?.from}
                selected={date}
                onSelect={(range) => {
                  setDate(range);
                  if (range?.from) setMonth(range.from);
                }}
                numberOfMonths={2}
              />
            </div>

            <div className="flex items-center justify-between gap-2 p-3">
              <PopoverClose asChild>
                <Button
                  onClick={onReset}
                  disabled={!date?.from || !date?.to}
                  className="h-9 w-9 p-0"
                  variant="outline"
                >
                  <RotateCcw className="size-4" />
                </Button>
              </PopoverClose>

              <div className="ml-auto flex items-center gap-2">
                <PopoverClose asChild>
                  <Button
                    onClick={() => pushToUrl(date)}
                    disabled={!date?.from || !date?.to}
                    className="h-9 min-w-24"
                  >
                    Apply
                  </Button>
                </PopoverClose>
              </div>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};
