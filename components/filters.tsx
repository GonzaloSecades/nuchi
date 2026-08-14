import { AccountFilter } from './account-filter';
import { DateFilter } from './date-filter';

export const Filters = () => {
  // Filters own URL state and only the data needed to render their controls.
  // Summary consumers react to those URL changes independently; keeping the
  // header free of summary loading state leaves it usable on every dashboard
  // page and while analytics refetches.
  return (
    <div className="flex flex-col items-center gap-y-2 lg:flex-row lg:gap-x-2 lg:gap-y-0">
      <AccountFilter />
      <DateFilter />
    </div>
  );
};
