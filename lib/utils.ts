import { clsx, type ClassValue } from 'clsx';
import { eachDayOfInterval, format, subDays } from 'date-fns';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function convertAmountFromMiliunits(amount: number) {
  return amount / 1000;
}

/**
 * Largest magnitude the API accepts for a milliunit amount.
 *
 * `transactions.amount` is a PostgreSQL bigint (migration 00005), whose range
 * is wider than this, but the API deliberately validates against JavaScript's
 * safe-integer limit so every amount it returns is exact in the browser. Values
 * past this bound are rejected with 400 rather than silently losing precision
 * here.
 */
export const MAX_SAFE_MILIUNITS = Number.MAX_SAFE_INTEGER;

export function convertAmountToMiliunits(amount: number) {
  return Math.round(amount * 1000);
}

/**
 * Reports whether a milliunit amount is exactly representable and within the
 * range the API accepts. Guards the submit paths (transaction form, CSV
 * import) so an out-of-range value fails with a field-level message instead of
 * a server 400 or, worse, a silently rounded number.
 */
export function isSafeMiliunitAmount(amountInMiliunits: number) {
  return (
    Number.isSafeInteger(amountInMiliunits) &&
    Math.abs(amountInMiliunits) <= MAX_SAFE_MILIUNITS
  );
}

export function formatCurrency(value: number) {
  return Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
  }).format(value);
}

export function calculatePercentageChange(current: number, previous: number) {
  if (previous === 0) {
    return previous === current ? 0 : 100;
  }
  return ((current - previous) / previous) * 100;
}

export function fillMissingDays(
  activeDays: {
    date: Date;
    income: number;
    expenses: number;
  }[],
  startDate: Date,
  endDate: Date
) {
  const allDays = eachDayOfInterval({ start: startDate, end: endDate });

  const byDay = new Map<
    string,
    { date: Date; income: number; expenses: number }
  >();

  for (const entry of activeDays) {
    const key = format(entry.date, 'yyyy-MM-dd');
    byDay.set(key, entry);
  }

  const transactionsByDay = allDays.map((day) => {
    const key = format(day, 'yyyy-MM-dd');
    const found = byDay.get(key);

    if (found) {
      return found;
    }

    return {
      date: day,
      income: 0,
      expenses: 0,
    };
  });

  return transactionsByDay;
}

type Period = {
  from: string | Date | undefined;
  to: string | Date | undefined;
};

export const formatDateRange = (period?: Period) => {
  const defaultTo = new Date();
  const defaultFrom = subDays(defaultTo, 30);

  if (!period?.from) {
    return `${format(defaultFrom, 'LLL dd')} - ${format(defaultTo, 'LLL dd, y')}`;
  }

  if (period.to) {
    return `${format(period.from, 'LLL dd')} - ${format(period.to, 'LLL dd, y')}`;
  }

  return format(period.from, 'LLL dd, y');
};

export const formatPercentage = (
  value: number,
  options: {
    addPrefix?: boolean;
  } = {
    addPrefix: false,
  }
) => {
  const result = new Intl.NumberFormat('en-US', {
    style: 'percent',
  }).format(value / 100);

  if (options.addPrefix && value > 0) {
    return `+${result}`;
  }

  return result;
};
