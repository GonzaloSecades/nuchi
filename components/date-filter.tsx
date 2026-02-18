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
import { ChevronDown, ChevronLeft, RotateCcw } from 'lucide-react';
import { useState } from 'react';
import { DateRange } from 'react-day-picker';
import { useMedia } from 'react-use';

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
import { cn, formatDateRange } from '@/lib/utils';

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
  const [activePreset, setActivePreset] = useState<PresetKey | null>(null);
  const [mobilePanel, setMobilePanel] = useState<'calendar' | 'presets'>(
    'calendar'
  );
  const isMobile = useMedia('(max-width: 640px)', false);

  const onOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);

    if (nextOpen) {
      // re-seed from URL-derived state every time the popover opens
      setDate(paramState);
      setMonth(paramState.from);
      setActivePreset(null);
      setMobilePanel('calendar');
    }
  };

  const pushToUrl = (
    dateRange: DateRange | undefined,
    mode: 'set' | 'clear' = 'set'
  ) => {
    const query =
      mode === 'clear'
        ? {
            accountId,
            from: null,
            to: null,
          }
        : {
            accountId,
            from: dateRange?.from ? format(dateRange.from, 'yyyy-MM-dd') : null,
            to: dateRange?.to ? format(dateRange.to, 'yyyy-MM-dd') : null,
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
    setActivePreset(null);
    pushToUrl(undefined, 'clear');
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
    setActivePreset(preset);

    if (preset !== 'allTime' && next.from) {
      setMonth(next.from);
    }

    if (isMobile) {
      setMobilePanel('calendar');
    }
  };

  const selectedDate = open ? date : paramState;

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
      <PopoverContent
        align={isMobile ? 'center' : 'start'}
        collisionPadding={isMobile ? 12 : 8}
        className={cn(
          'overflow-hidden rounded-xl p-0',
          isMobile
            ? 'w-[min(23rem,calc(100vw-1rem))] max-w-[calc(100vw-1rem)]'
            : 'w-auto max-w-[calc(100vw-2rem)]'
        )}
      >
        <div className={cn('flex', isMobile ? 'w-full' : 'w-fit')}>
          {/* Presets column */}
          <aside
            className={cn(
              'bg-muted/20 shrink-0 p-3',
              isMobile
                ? 'w-full border-b'
                : 'w-42.5 border-r',
              isMobile && mobilePanel !== 'presets' && 'hidden'
            )}
          >
            {isMobile ? (
              <div className="space-y-3">
                <div className="flex items-center">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => setMobilePanel('calendar')}
                    className="-ml-2 h-8 px-2 text-sm font-normal"
                  >
                    <ChevronLeft className="mr-1 size-4" />
                    Calendar
                  </Button>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  {PRESETS.map((preset) => (
                    <Button
                      key={preset.key}
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => onPresetSelect(preset.key)}
                      className={cn(
                        'h-9 w-full justify-center rounded-md border px-2 text-sm font-normal',
                        activePreset === preset.key
                          ? 'border-primary bg-primary text-primary-foreground hover:bg-primary/90'
                          : 'border-border bg-background hover:bg-muted'
                      )}
                    >
                      {preset.label}
                    </Button>
                  ))}
                </div>
              </div>
            ) : null}
            {!isMobile ? (
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
            ) : null}
          </aside>

          {/* Calendar + actions */}
          <div
            className={cn(
              'flex-1',
              isMobile && mobilePanel === 'presets' && 'hidden'
            )}
          >
            <div className={cn('border-b p-3', isMobile ? 'w-full' : 'w-fit')}>
              {isMobile ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => setMobilePanel('presets')}
                  className="-ml-2 mb-2 h-8 px-2 text-sm font-normal"
                >
                  <ChevronLeft className="mr-1 size-4" />
                  Presets
                </Button>
              ) : null}
              <Calendar
                disabled={false}
                autoFocus
                mode="range"
                month={month}
                onMonthChange={setMonth}
                defaultMonth={date?.from}
                selected={selectedDate}
                onSelect={(range) => {
                  setDate(range);
                  setActivePreset(null);
                  if (range?.from) setMonth(range.from);
                }}
                numberOfMonths={isMobile ? 1 : 2}
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
