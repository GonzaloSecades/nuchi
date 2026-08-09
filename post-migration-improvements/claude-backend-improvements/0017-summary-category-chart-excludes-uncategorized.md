# 0017 — The summary category chart silently excludes uncategorized spending

- **Migration ticket:** #48 (https://github.com/GonzaloSecades/nuchi/issues/48)
- **Area:** api / product
- **Priority guess:** medium

## How it was migrated

`GetCategorySpending` inner-joins `categories`, so the breakdown contains only
expenses that have a category. `expensesAmount` comes from a separate query
with no such join and therefore includes everything.

Consequence: **the category chart does not sum to the expenses total** whenever
the user has any uncategorized spending. A user with 155,000 in expenses, 1,000
of it uncategorized, sees a chart totalling 154,000 with nothing explaining the
gap.

Ported faithfully. Legacy does exactly this — `innerJoin(categories, ...)` in
`app/api/[[...route]]/summary.ts`.

## Why it was done this way

Parity freezes observable behavior, and this is observable: adding an
"Uncategorized" bucket would change what the dashboard renders for most users
on day one. The fixtures state the rule plainly ("Category breakdown includes
only negative transactions with a category"), so it is frozen behavior rather
than an oversight to quietly correct mid-migration.

It is also genuinely a product decision rather than a bug to fix by reflex.
Both readings are defensible:

- **Current behavior:** the chart answers "where does my categorized spending
  go", and an Uncategorized slice is noise that crowds out real categories —
  especially for a user who has not set categories up yet, where it would be
  100% of the chart.
- **The alternative:** a chart that claims to break down expenses should
  account for all of them, and the invisible gap is worse than an ugly slice
  because the user cannot see that anything is missing.

## The concern

The numbers on one screen disagree and nothing says why. That is the kind of
discrepancy that erodes trust in a finance app more than a missing feature
would, because the user cannot tell whether the total or the chart is wrong.

It also hides a real prompt: uncategorized spending is exactly what the user
should be nudged to categorize, and the current design makes it invisible
rather than actionable.

Interacts with the `Other` bucket: with more than three categories the chart
already aggregates a tail, so a user seeing "Other" may reasonably assume it
covers uncategorized spending too. It does not.

## Proposed improvement

After parity, make the gap visible. In rough order of increasing effort:

1. **Add an explicit `Uncategorized` entry** computed as
   `expensesAmount - sum(categorized)`, so the chart always reconciles with the
   total. Needs a decision on whether it participates in the top-3 ranking or
   is pinned last — pinned last is probably right, since it is a prompt rather
   than a category.
2. **Surface it as a call to action** rather than a slice: "12,500 uncategorized
   — categorize now", linking to a filtered transaction list.
3. Revisit the fixed top-3 cap while there, which is a display choice hardcoded
   in the API. Returning the full breakdown and letting the client decide how
   many to show would make the endpoint less opinionated.

Whichever is chosen, the contract's `SummaryCategory` description and the
product documentation both need updating alongside it.
