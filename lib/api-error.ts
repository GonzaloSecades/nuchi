/**
 * API Error Handling Utilities
 *
 * This module provides a standardized way to handle API errors across the application.
 * It captures detailed error information and provides a single point for:
 * - Error logging (Sentry, LogRocket, etc.)
 * - Error formatting and normalization
 * - Future integrations (analytics, alerting, etc.)
 *
 * Usage:
 * ------
 * ```typescript
 * if (!response.ok) {
 *   throw await createApiError(response, 'accounts');
 * }
 * ```
 */

/**
 * Structured error details for API failures
 */
export interface ApiErrorDetails {
  status: number;
  statusText: string;
  url: string;
  resource: string;
  timestamp: string;
  errorData: unknown;
}

/**
 * Custom error class for API-related errors
 * Extends Error with additional structured metadata
 */
export class ApiError extends Error {
  public readonly details: ApiErrorDetails;

  constructor(message: string, details: ApiErrorDetails) {
    super(message);
    this.name = 'ApiError';
    this.details = details;

    // Maintains proper stack trace for where error was thrown (V8 engines)
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, ApiError);
    }
  }

  /**
   * Check if error is a specific HTTP status
   */
  isStatus(status: number): boolean {
    return this.details.status === status;
  }

  /**
   * Check if error is a client error (4xx)
   */
  isClientError(): boolean {
    return this.details.status >= 400 && this.details.status < 500;
  }

  /**
   * Check if error is a server error (5xx)
   */
  isServerError(): boolean {
    return this.details.status >= 500;
  }

  /**
   * Check if error is unauthorized (401)
   */
  isUnauthorized(): boolean {
    return this.details.status === 401;
  }

  /**
   * Check if error is forbidden (403)
   */
  isForbidden(): boolean {
    return this.details.status === 403;
  }

  /**
   * Check if error is not found (404)
   */
  isNotFound(): boolean {
    return this.details.status === 404;
  }
}

/**
 * Creates a standardized ApiError from a failed Response
 *
 * @param response - The failed fetch Response object
 * @param resource - A descriptive name for the resource (e.g., 'accounts', 'transactions')
 * @returns Promise<ApiError> - A structured error with all relevant details
 *
 * @remarks
 * This function clones the response before reading the body, so the original
 * response object remains usable if needed elsewhere. The clone is used to
 * safely attempt JSON parsing without affecting the original response stream.
 *
 * @example
 * ```typescript
 * const response = await client.api.accounts.$get();
 * if (!response.ok) {
 *   throw await createApiError(response, 'accounts');
 * }
 * ```
 */
export async function createApiError(response: Response, resource: string): Promise<ApiError> {
  // Clone the response before reading the body to preserve the original stream
  // This allows the caller to read the response body again if needed
  const clonedResponse = response.clone();

  // Safely attempt to parse error data from the cloned response body
  const errorData = await clonedResponse.json().catch(() => null);

  const details: ApiErrorDetails = {
    status: response.status,
    statusText: response.statusText,
    url: response.url,
    resource,
    timestamp: new Date().toISOString(),
    errorData,
  };

  const message = `Failed to fetch ${resource}: ${response.status} ${response.statusText}`;

  const error = new ApiError(message, details);

  // 🔌 LOGGING INTEGRATION POINT
  // Uncomment and configure when ready to add logging services
  // --------------------------------------------------------
  // logError(error);

  return error;
}

/**
 * Logs an error to configured logging services
 *
 * This is the central point for all error logging.
 * Add your Sentry, LogRocket, or custom logging here.
 *
 * @param error - The ApiError to log
 */
export function logError(error: ApiError): void {
  // Always log to console in development
  if (process.env.NODE_ENV === 'development') {
    console.error('[API Error]', {
      message: error.message,
      details: error.details,
      stack: error.stack,
    });
  }

  // 🔌 SENTRY INTEGRATION
  // --------------------------------------------------------
  // import * as Sentry from '@sentry/nextjs';
  //
  // Sentry.captureException(error, {
  //   tags: {
  //     resource: error.details.resource,
  //     status: error.details.status,
  //   },
  //   extra: {
  //     url: error.details.url,
  //     errorData: error.details.errorData,
  //     timestamp: error.details.timestamp,
  //   },
  // });

  // 🔌 LOGROCKET INTEGRATION
  // --------------------------------------------------------
  // import LogRocket from 'logrocket';
  //
  // LogRocket.captureException(error, {
  //   tags: { resource: error.details.resource },
  //   extra: error.details,
  // });

  // 🔌 CUSTOM ANALYTICS / ALERTING
  // --------------------------------------------------------
  // await fetch('/api/log', {
  //   method: 'POST',
  //   body: JSON.stringify(error.details),
  // });
}

/**
 * Type guard to check if an error is an ApiError
 *
 * @param error - Any error object
 * @returns boolean - True if error is an ApiError
 *
 * @example
 * ```typescript
 * if (isApiError(error) && error.isUnauthorized()) {
 *   redirect('/sign-in');
 * }
 * ```
 */
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
