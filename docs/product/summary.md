# Summary — Product Behavior

What the dashboard shows, how each number is worked out, and the two places
where the figures look wrong but aren't.

Written for product decisions and support answers. For endpoint shapes see
[the technical reference](../api/summary.md).

## What the dashboard shows

One request produces everything on the overview screen:

| Figure | Meaning |
| --- | --- |
| Remaining | Income minus expenses for the period |
| Income | Everything received |
| Expenses | Everything spent, as a positive number |
| Change % | How each of the three compares with the previous period |
| Category chart | Where the categorized spending went |
| Daily chart | Income and expenses per day across the period |

## The period

Same filters as the transaction list, and deliberately so — one date control
drives both, so the list and the charts always describe the same window.

Default is **the last 30 days**. Both ends are inclusive, and the longest
allowed range is 366 days.

## "Compared to previous period" means the period immediately before

This is the single most likely support question.

The comparison is against **the same number of days, immediately preceding** the
selected range — not the previous calendar month.

Looking at 1–31 March compares against 29 January – 28 February, not "February"
as a month. Viewing the last 7 days compares against the 7 days before that. A
user expecting month-over-month will read the percentage as wrong when it is
working as designed.

### Why a change can read 100%

When the previous period had **nothing** and the current one has **something**,
the change is reported as **100%**, not infinity and not a blank. There is no
meaningful percentage increase from zero, and 100% is the app's way of saying
"all of this is new".

When both periods are empty the change is **0%**, not 100%.

So a brand-new user's first month shows +100% on income and expenses, and a
user with no activity at all sees 0% everywhere. Both are correct.

## The category chart does not add up to the expenses total

The second likely support question, and a real quirk rather than a bug in the
arithmetic.

The chart includes only expenses that **have a category**. Uncategorized
spending still counts toward the Expenses figure but does not appear in the
chart, and nothing on screen explains the difference.

A user with 155,000 in expenses, 1,000 of it uncategorized, sees a chart
totalling 154,000. The gap is invisible.

This is inherited from the previous implementation and was kept deliberately
during the migration so behavior did not change mid-flight. It is queued to be
addressed — most likely by showing uncategorized spending explicitly, so the
chart reconciles with the total and the user gets a nudge to categorize.

Income never appears in the category chart either, even when the transaction
has a category. The chart is about spending only.

### "Other"

Only the three largest categories are shown individually. Everything else is
summed into a single **Other** slice.

With exactly three categories there is no Other slice. And note Other covers
only *smaller categories* — it does **not** include uncategorized spending,
which is a reasonable thing for a user to assume and is not the case.

## The daily chart always covers every day

Days with no transactions appear as zero rather than being skipped, so the
chart draws a continuous line across the period. Without this, a gap would be
drawn as a straight line between two active days and imply activity that never
happened.

A period with no transactions at all still returns a full set of zero days, so
the chart renders flat rather than empty.

## Filtering by one account

Narrows every figure on the screen — totals, changes, both charts.

Filtering by an account that isn't the user's own returns zeros rather than an
error. The app does not confirm or deny that another user's account exists.

## What this deliberately cannot do yet

1. **Uncategorized spending is invisible in the chart** (above). The most
   user-visible item here.
2. **Only the top three categories** are broken out, and that is fixed in the
   API rather than a display preference.
3. **No month-over-month comparison** — the comparison window is always
   length-based.
4. **No caching.** Every dashboard load recomputes from scratch. Fine at
   personal-finance scale; something to revisit as histories grow.
5. **Day boundaries are UTC**, so a transaction near midnight can land on a
   neighbouring day relative to the user's local date.

## Related

- [Technical reference](../api/summary.md)
- [Transactions — product behavior](transactions.md) — the filters, milliunits,
  and sign conventions this builds on
- `post-migration-improvements/claude-backend-improvements/0017-...` — the
  uncategorized-spending gap
