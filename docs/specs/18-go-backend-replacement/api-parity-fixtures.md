# Current API Parity Fixtures

Parent ticket: [#34 Document current API parity fixtures](https://github.com/GonzaloSecades/nuchi/issues/34)

Parent spec: [#18 Spec Go backend replacement for Hono/Drizzle/Neon](https://github.com/GonzaloSecades/nuchi/issues/18)

Related epic: [#29 OpenAPI contract epic](https://github.com/GonzaloSecades/nuchi/issues/29)

Spec source: [Go backend replacement spec](./spec.md)

This document captures the current Hono API behavior that the Go replacement must preserve unless the replacement spec names an intentional product change.

Source files:

- `db/schema.ts`
- `app/api/[[...route]]/route.ts`
- `app/api/[[...route]]/accounts.ts`
- `app/api/[[...route]]/categories.ts`
- `app/api/[[...route]]/transactions.ts`
- `app/api/[[...route]]/summary.ts`
- `lib/chunk-items.ts`
- `lib/transaction-limits.ts`
- `lib/transaction-route-utils.ts`
- `lib/transaction-import.ts`
- `lib/utils.ts`
- `proxy.ts`

## Shared Behavior

All resource routes are mounted under `/api`:

- `/api/accounts`
- `/api/categories`
- `/api/transactions`
- `/api/summary`

Successful app resource responses use a `{ "data": ... }` envelope.

Unauthenticated Hono handlers return:

```json
{ "error": "Unauthorized" }
```

with HTTP `401`.

Most hand-written route errors use one of these shapes:

```json
{ "error": "Missing id" }
```

```json
{
  "error": {
    "code": "DB_ERROR",
    "message": "DatabaseError - Failed to fetch accounts"
  }
}
```

Validation errors from `@hono/zod-validator` return HTTP `400` with the package default shape:

```json
{
  "success": false,
  "error": {},
  "data": {}
}
```

The real `error` value is the serialized Zod error for the target validator.

## Auth Redirects And API Auth Errors

Clerk middleware in `proxy.ts` protects all non-static routes and all API routes. Only these routes are public:

- `/sign-in(.*)?`
- `/sign-up(.*)?`

Browser navigation to protected app pages is handled by Clerk and redirects to sign-in when there is no session.

API route handlers also check `getAuth(c)?.userId`. When a Hono handler runs without a user, it returns HTTP `401`:

```json
{ "error": "Unauthorized" }
```

The Go replacement should keep API errors explicit and keep browser redirects separate from JSON API behavior.

## Accounts

### `GET /api/accounts`

Lists accounts owned by the authenticated user.

Response `200`:

```json
{
  "data": [
    {
      "id": "acc_1",
      "name": "Cash"
    }
  ]
}
```

### `GET /api/accounts/:id`

Gets one owned account.

Response `200`:

```json
{
  "data": {
    "id": "acc_1",
    "name": "Cash"
  }
}
```

Known errors:

- `400`: `{ "error": "Missing id" }`
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Account not found" }`

### `POST /api/accounts`

Creates an account for the authenticated user.

Request:

```json
{ "name": "Cash" }
```

Response `200`:

```json
{
  "data": {
    "id": "acc_1",
    "plaidId": null,
    "name": "Cash",
    "userId": "user_1"
  }
}
```

Known errors:

- `400`: Zod validation error
- `401`: `{ "error": "Unauthorized" }`
- `409`: duplicate account name for the same user

Duplicate response:

```json
{
  "error": {
    "code": "DUPLICATE_ACCOUNT_NAME",
    "message": "You already have an account with this name.",
    "constraint": "accounts_user_id_name_uniq"
  }
}
```

### `PATCH /api/accounts/:id`

Updates an owned account.

Request:

```json
{ "name": "Checking" }
```

Response `200` uses the same full account row shape as create.

Known errors:

- `400`: `{ "error": "Missing account id" }` or Zod validation error
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Account not found" }`
- `409`: same duplicate response as create

### `DELETE /api/accounts/:id`

Deletes one owned account. Account deletion cascades transactions through the database foreign key.

Response `200`:

```json
{
  "data": {
    "id": "acc_1"
  }
}
```

Known errors:

- `400`: `{ "error": "Missing account id" }`
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Account not found" }`

### `POST /api/accounts/bulk-delete`

Deletes owned accounts matching the requested IDs.

Request:

```json
{ "ids": ["acc_1", "acc_2"] }
```

Response `200`:

```json
{
  "data": [
    { "id": "acc_1" },
    { "id": "acc_2" }
  ]
}
```

Validation requires at least one non-empty ID. Missing or unowned IDs are ignored by the delete query rather than returned as `404`.

## Categories

Categories mirror account behavior, scoped by authenticated user. Account and category names use Postgres `citext`, so duplicate-name checks are case-insensitive per user.

### `GET /api/categories`

Response `200`:

```json
{
  "data": [
    {
      "id": "cat_1",
      "name": "Groceries"
    }
  ]
}
```

### `GET /api/categories/:id`

Response `200`:

```json
{
  "data": {
    "id": "cat_1",
    "name": "Groceries"
  }
}
```

Known errors:

- `400`: `{ "error": "Missing id" }`
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Category not found" }`

### `POST /api/categories`

Request:

```json
{ "name": "Groceries" }
```

Response `200`:

```json
{
  "data": {
    "id": "cat_1",
    "plaidId": null,
    "name": "Groceries",
    "userId": "user_1"
  }
}
```

Duplicate response `409`:

```json
{
  "error": {
    "code": "DUPLICATE_CATEGORY_NAME",
    "message": "You already have a category with this name.",
    "constraint": "categories_user_id_name_uniq"
  }
}
```

### `PATCH /api/categories/:id`

Request:

```json
{ "name": "Food" }
```

Response `200` uses the same full category row shape as create.

Known errors:

- `400`: `{ "error": "Missing category id" }` or Zod validation error
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Category not found" }`

Current category update catches database errors as `500`; unlike accounts, duplicate update is not mapped to `409`.

### `DELETE /api/categories/:id`

Deletes one owned category. Category deletion sets matching transaction `categoryId` values to `null` through the database foreign key.

Response `200`:

```json
{
  "data": {
    "id": "cat_1"
  }
}
```

Known errors:

- `400`: `{ "error": "Missing category id" }`
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Category not found" }`

### `POST /api/categories/bulk-delete`

Request:

```json
{ "ids": ["cat_1", "cat_2"] }
```

Response `200`:

```json
{
  "data": [
    { "id": "cat_1" },
    { "id": "cat_2" }
  ]
}
```

Validation requires at least one non-empty ID. Missing or unowned IDs are ignored by the delete query rather than returned as `404`.

## Transactions

Transaction amounts are stored as signed integer milliunits. Positive values are income. Negative values are expenses.

Example: `10.5` in the UI is stored as `10500`; `-10.5` is stored as `-10500`.

Transactions are owned through their required account. Category is optional, but when provided it must be owned by the same authenticated user.

### `GET /api/transactions`

Query parameters:

- `from`: optional `yyyy-MM-dd`
- `to`: optional `yyyy-MM-dd`
- `accountId`: optional account ID

Date behavior:

- Default `to` is current server time when `to` is omitted.
- Default `from` is 30 days before current server time when `from` is omitted. A provided `to` does not move the default `from`.
- `from` is parsed as the start of that calendar day.
- `to` is parsed as the end of that calendar day.
- Filtering is inclusive: `date >= from` and `date <= to`.
- Maximum inclusive date range is 366 days.
- Invalid dates return HTTP `400`.
- `accountId` filters transactions by that account. If the account is missing or unowned, the list is empty because the query still requires an owned joined account.

Response `200`:

```json
{
  "data": [
    {
      "id": "txn_1",
      "date": "2026-06-30T00:00:00.000Z",
      "category": "Groceries",
      "categoryId": "cat_1",
      "payee": "Market",
      "amount": -12500,
      "notes": "weekly shop",
      "account": "Cash",
      "accountId": "acc_1"
    }
  ]
}
```

Rows are sorted by transaction date descending.

Date query errors:

```json
{
  "error": {
    "code": "INVALID_QUERY",
    "message": "from and to must use yyyy-MM-dd dates."
  }
}
```

```json
{
  "error": {
    "code": "INVALID_QUERY",
    "message": "from must be less than or equal to to."
  }
}
```

```json
{
  "error": {
    "code": "INVALID_QUERY",
    "message": "Date range cannot exceed 366 days."
  }
}
```

### `GET /api/transactions/:id`

Gets one transaction joined through an owned account.

Response `200`:

```json
{
  "data": {
    "id": "txn_1",
    "date": "2026-06-30T00:00:00.000Z",
    "categoryId": "cat_1",
    "payee": "Market",
    "amount": -12500,
    "notes": "weekly shop",
    "accountId": "acc_1"
  }
}
```

Known errors:

- `400`: `{ "error": "Missing id" }`
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Transaction not found" }`

### `POST /api/transactions`

Request:

```json
{
  "amount": -12500,
  "payee": "Market",
  "notes": "weekly shop",
  "date": "2026-06-30",
  "accountId": "acc_1",
  "categoryId": "cat_1"
}
```

`categoryId` may be `null` or omitted.

Response `200`:

```json
{
  "data": {
    "id": "txn_1",
    "amount": -12500,
    "payee": "Market",
    "notes": "weekly shop",
    "date": "2026-06-30T00:00:00.000Z",
    "accountId": "acc_1",
    "categoryId": "cat_1"
  }
}
```

Known errors:

- `400`: Zod validation error
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Account not found" }` or `{ "error": "Category not found" }`
- `429`: transaction mutation rate limit

### `PATCH /api/transactions/:id`

Request shape is the same as create.

Response `200` uses the same full transaction row shape as create.

Known errors:

- `400`: `{ "error": "Missing Transaction id" }` or Zod validation error
- `401`: `{ "error": "Unauthorized" }`
- `404`: missing owned transaction, account, or category
- `429`: transaction mutation rate limit

### `DELETE /api/transactions/:id`

Deletes one transaction joined through an owned account.

Response `200`:

```json
{
  "data": {
    "id": "txn_1"
  }
}
```

Known errors:

- `400`: `{ "error": "Missing transaction id" }`
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Transaction not found" }`
- `429`: transaction mutation rate limit

### `POST /api/transactions/bulk-create`

Creates many transactions. The server validates the full JSON array and all referenced owned accounts/categories before inserting.

Request:

```json
[
  {
    "amount": -12500,
    "payee": "Market",
    "date": "2026-06-30",
    "accountId": "acc_1",
    "categoryId": "cat_1"
  },
  {
    "amount": 500000,
    "payee": "Salary",
    "date": "2026-06-30",
    "accountId": "acc_1",
    "categoryId": null
  }
]
```

Response `200` returns every inserted row:

```json
{
  "data": [
    {
      "id": "txn_1",
      "amount": -12500,
      "payee": "Market",
      "notes": null,
      "date": "2026-06-30T00:00:00.000Z",
      "accountId": "acc_1",
      "categoryId": "cat_1"
    },
    {
      "id": "txn_2",
      "amount": 500000,
      "payee": "Salary",
      "notes": null,
      "date": "2026-06-30T00:00:00.000Z",
      "accountId": "acc_1",
      "categoryId": null
    }
  ]
}
```

Limits:

- JSON array minimum: 1 transaction
- JSON array maximum: 500 transactions
- Numeric `Content-Length` over 1,000,000 bytes returns `413`; absent or non-numeric `Content-Length` is not rejected by this guard.

Known errors:

- `400`: Zod validation error
- `401`: `{ "error": "Unauthorized" }`
- `404`: `{ "error": "Account not found" }` or `{ "error": "Category not found" }`
- `413`: request body too large
- `429`: transaction mutation rate limit

Request body too large response:

```json
{
  "error": {
    "code": "REQUEST_BODY_TOO_LARGE",
    "message": "Request body is too large."
  }
}
```

### `POST /api/transactions/bulk-delete`

Deletes owned transactions matching requested IDs.

Request:

```json
{ "ids": ["txn_1", "txn_2"] }
```

Response `200`:

```json
{
  "data": [
    { "id": "txn_1" },
    { "id": "txn_2" }
  ]
}
```

Limits:

- `ids` minimum: 1 non-empty ID
- `ids` maximum: 500 IDs
- Numeric `Content-Length` over 100,000 bytes returns `413`; absent or non-numeric `Content-Length` is not rejected by this guard.

Missing or unowned IDs are ignored by the delete query rather than returned as `404`.

Known errors:

- `400`: Zod validation error
- `401`: `{ "error": "Unauthorized" }`
- `413`: request body too large
- `429`: transaction mutation rate limit

## Transaction Mutation Rate Limit

Transaction create, bulk-create, patch, delete, and bulk-delete are limited per authenticated user and action.

Current limits:

- Window: 60 seconds
- Maximum requests: 60 per window
- Key: `${userId}:${action}`

Response `429`:

```json
{
  "error": {
    "code": "TRANSACTION_MUTATION_RATE_LIMITED",
    "message": "Too many transaction mutations. Please try again later."
  }
}
```

The response includes a `Retry-After` header in seconds.

## CSV Import To Bulk Create

CSV import is client-side parsing followed by `POST /api/transactions/bulk-create`.

CSV upload uses `react-papaparse` and lets the user map columns to:

- `amount`
- `date`
- `payee`

All three mapped fields are required before continuing. Each field can only be mapped to one CSV column.

Parser input row shape:

```json
{
  "amount": "-12.5",
  "date": "2026-06-30 15:04:05",
  "payee": "Market"
}
```

Parser output row shape:

```json
{
  "amount": -12500,
  "date": "2026-06-30",
  "payee": "Market"
}
```

CSV validation:

- `amount` is trimmed and parsed with `Number(...)`; it must be finite.
- `date` must exactly match `yyyy-MM-dd HH:mm:ss`.
- `payee` is trimmed and required.
- Invalid rows block continuing and return row-level client errors.

Import conversion:

- Amounts are converted with `Math.round(amount * 1000)`.
- Dates are reformatted from `yyyy-MM-dd HH:mm:ss` to `yyyy-MM-dd`.
- The user selects one account after CSV validation.
- The selected `accountId` is added to every imported row.
- `categoryId` and `notes` are not set by CSV import.
- Rows are posted in chunks of 500 transactions.

Bulk-create request produced by CSV import:

```json
[
  {
    "amount": -12500,
    "date": "2026-06-30",
    "payee": "Market",
    "accountId": "acc_1"
  }
]
```

If the parsed CSV data has no valid rows, `chunkItems([], 500)` produces no API requests after account selection.

## Summary

### `GET /api/summary`

Query parameters:

- `from`: optional `yyyy-MM-dd`
- `to`: optional `yyyy-MM-dd`
- `accountId`: optional account ID

Date behavior matches `GET /api/transactions`:

- Defaults match transaction list: omitted `to` uses current server time, and omitted `from` uses 30 days before current server time. A provided `to` does not move the default `from`.
- `from` is start of day.
- `to` is end of day.
- Filtering is inclusive.
- Maximum inclusive date range is 366 days.
- Invalid dates return HTTP `400`.

Response `200`:

```json
{
  "data": {
    "remainingAmount": 487500,
    "remainingChange": 12.5,
    "incomeAmount": 500000,
    "incomeChange": 100,
    "expensesAmount": 12500,
    "expensesChange": -20,
    "categories": [
      {
        "name": "Groceries",
        "value": 12500
      }
    ],
    "days": [
      {
        "date": "2026-06-30T00:00:00.000Z",
        "income": 500000,
        "expenses": 12500
      }
    ]
  }
}
```

Summary calculations:

- `remainingAmount`: sum of signed transaction amounts.
- `incomeAmount`: sum of amounts greater than or equal to `0`.
- `expensesAmount`: sum of absolute values for amounts less than `0`.
- Change fields compare the selected period with the immediately preceding period of the same length.
- `calculatePercentageChange(0, 0)` returns `0`.
- `calculatePercentageChange(current, 0)` returns `100` when current is non-zero.
- Category breakdown includes only negative transactions with a category.
- Category values are absolute expense sums in milliunits.
- The top 3 categories are returned as-is.
- If more than 3 expense categories exist, the rest are grouped into `{ "name": "Other", "value": ... }`.
- `days` includes every calendar day in the selected range. Missing days are filled with `income: 0` and `expenses: 0`.

Known errors:

- `400`: invalid date query
- `401`: `{ "error": "Unauthorized" }`
- `500`: database error

Invalid date query responses are the same shape and messages as transaction list date errors.

## Error Status Inventory

Known current statuses to preserve or intentionally change in OpenAPI:

- `400`: missing route IDs, invalid query dates, Zod request validation failures.
- `401`: missing authenticated user.
- `404`: missing owned account, category, or transaction for single-resource operations and transaction reference validation.
- `409`: duplicate account create/update and duplicate category create.
- `413`: oversized transaction bulk-create or bulk-delete request bodies.
- `429`: transaction mutation rate limit.
- `500`: database catch blocks.

Current mismatches worth deciding in OpenAPI:

- Category duplicate update currently returns `500`, while duplicate create returns `409`.
- Bulk deletes for accounts, categories, and transactions ignore missing/unowned IDs instead of returning `404`.
- Transaction bulk-delete is not all-or-error; transaction bulk-create is all-or-error for validation and reference ownership.
