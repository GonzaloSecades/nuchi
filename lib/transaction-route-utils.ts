import {
  differenceInCalendarDays,
  endOfDay,
  format,
  isValid,
  parse,
} from 'date-fns';

export {
  MAX_BULK_CREATE_BODY_BYTES,
  MAX_BULK_CREATE_TRANSACTIONS,
  MAX_BULK_DELETE_BODY_BYTES,
  MAX_BULK_DELETE_TRANSACTIONS,
} from './transaction-limits';

const MUTATION_RATE_LIMIT_WINDOW_MS = 60_000;
const MUTATION_RATE_LIMIT_MAX_REQUESTS = 60;
const MUTATION_RATE_LIMIT_CLEANUP_INTERVAL_MS = 300_000;

const mutationRequestsByKey = new Map<string, number[]>();
let lastMutationRateLimitCleanup = 0;

type ParseStrictDateOptions = {
  boundary?: 'start' | 'end';
};

type DateRangeQueryOptions = {
  from: string | undefined;
  to: string | undefined;
  defaultFrom: Date;
  defaultTo: Date;
  maxDays?: number;
};

type DateRangeQueryResult =
  | {
      ok: true;
      startDate: Date;
      endDate: Date;
    }
  | {
      ok: false;
      status: 400;
      error: {
        code: 'INVALID_QUERY';
        message: string;
      };
    };

export function parseStrictDate(
  value: string | undefined,
  options: ParseStrictDateOptions = {}
) {
  if (!value) {
    return null;
  }

  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return null;
  }

  const parsed = parse(value, 'yyyy-MM-dd', new Date());

  if (!isValid(parsed) || format(parsed, 'yyyy-MM-dd') !== value) {
    return null;
  }

  return options.boundary === 'end' ? endOfDay(parsed) : parsed;
}

export function parseDateRangeQuery({
  from,
  to,
  defaultFrom,
  defaultTo,
  maxDays = 366,
}: DateRangeQueryOptions): DateRangeQueryResult {
  const parsedFrom = parseStrictDate(from);
  const parsedTo = parseStrictDate(to, { boundary: 'end' });

  if ((from && !parsedFrom) || (to && !parsedTo)) {
    return {
      ok: false,
      status: 400,
      error: {
        code: 'INVALID_QUERY',
        message: 'from and to must use yyyy-MM-dd dates.',
      },
    };
  }

  const startDate = parsedFrom ?? defaultFrom;
  const endDate = parsedTo ?? defaultTo;

  if (startDate > endDate) {
    return {
      ok: false,
      status: 400,
      error: {
        code: 'INVALID_QUERY',
        message: 'from must be less than or equal to to.',
      },
    };
  }

  if (differenceInCalendarDays(endDate, startDate) + 1 > maxDays) {
    return {
      ok: false,
      status: 400,
      error: {
        code: 'INVALID_QUERY',
        message: `Date range cannot exceed ${maxDays} days.`,
      },
    };
  }

  return {
    ok: true,
    startDate,
    endDate,
  };
}

function cleanupMutationRateLimit(now: number) {
  if (
    now - lastMutationRateLimitCleanup <
    MUTATION_RATE_LIMIT_CLEANUP_INTERVAL_MS
  ) {
    return;
  }

  const windowStart = now - MUTATION_RATE_LIMIT_WINDOW_MS;

  for (const [key, timestamps] of mutationRequestsByKey) {
    const recentRequests = timestamps.filter(
      (timestamp) => timestamp > windowStart
    );

    if (recentRequests.length > 0) {
      mutationRequestsByKey.set(key, recentRequests);
    } else {
      mutationRequestsByKey.delete(key);
    }
  }

  lastMutationRateLimitCleanup = now;
}

export function checkTransactionMutationRateLimit(
  userId: string,
  action: string
) {
  const key = `${userId}:${action}`;
  const now = Date.now();
  cleanupMutationRateLimit(now);

  const windowStart = now - MUTATION_RATE_LIMIT_WINDOW_MS;
  const recentRequests = (mutationRequestsByKey.get(key) ?? []).filter(
    (timestamp) => timestamp > windowStart
  );

  if (recentRequests.length >= MUTATION_RATE_LIMIT_MAX_REQUESTS) {
    mutationRequestsByKey.set(key, recentRequests);

    return {
      allowed: false,
      retryAfterMs: recentRequests[0] + MUTATION_RATE_LIMIT_WINDOW_MS - now,
    } as const;
  }

  recentRequests.push(now);
  mutationRequestsByKey.set(key, recentRequests);

  return {
    allowed: true,
  } as const;
}

export function isContentLengthTooLarge(
  contentLength: string | undefined,
  maxBytes: number
) {
  if (!contentLength) {
    return false;
  }

  const parsed = Number(contentLength);

  if (!Number.isFinite(parsed)) {
    return false;
  }

  return parsed > maxBytes;
}
