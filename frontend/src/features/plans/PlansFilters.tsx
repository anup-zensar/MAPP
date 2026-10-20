import { ChevronDown, SlidersHorizontal } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useId } from 'react';
import type { PlanListFilters } from './listPreferences';
import styles from './PlansFilters.module.css';

export interface PlansFiltersProps {
  filters: PlanListFilters;
  /** Every year with a plan in the team scope, newest first (PlanList.years). */
  years: number[];
  onChange: (next: Partial<PlanListFilters>) => void;
}

/** How many of the menu's filters differ from their defaults: the "(n)" of **Filters (n)**. */
function filtersInUse(filters: PlanListFilters): number {
  return (filters.year !== null ? 1 : 0) + (filters.scope !== 'all' ? 1 : 0);
}

/**
 * **Filters (n)** of the Plans toolbar (design system §5.2, list toolbar): a button that opens the
 * less-used filters — the year and whose plans to show — so the toolbar stays one row. Each
 * change applies at once; the chips under the toolbar show and remove them.
 */
export function PlansFilters({ filters, years, onChange }: PlansFiltersProps) {
  const id = useId();
  const count = filtersInUse(filters);
  const yearOptions =
    filters.year !== null && !years.includes(filters.year) ? [...years, filters.year] : years;

  return (
    <Popover.Root>
      <Popover.Trigger className={styles.trigger}>
        <SlidersHorizontal aria-hidden="true" />
        {count > 0 ? `Filters (${count})` : 'Filters'}
        <ChevronDown aria-hidden="true" />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content className={styles.menu} align="start" sideOffset={6} aria-label="Filters">
          <div className={styles.field}>
            <label htmlFor={`${id}-year`} className={styles.label}>
              Year
            </label>
            <select
              id={`${id}-year`}
              className={styles.select}
              value={filters.year === null ? '' : String(filters.year)}
              onChange={(event) =>
                onChange({ year: event.target.value ? Number(event.target.value) : null })
              }
            >
              <option value="">All years</option>
              {yearOptions.map((year) => (
                <option key={year} value={year}>
                  {year}
                </option>
              ))}
            </select>
          </div>
          <fieldset className={styles.field}>
            <legend className={styles.label}>Show</legend>
            {(
              [
                ['all', 'All plans'],
                ['myTeams', 'My teams’ plans'],
              ] as const
            ).map(([value, label]) => (
              <label key={value} className={styles.radio}>
                <input
                  type="radio"
                  name={`${id}-scope`}
                  value={value}
                  checked={filters.scope === value}
                  onChange={() => onChange({ scope: value })}
                />
                {label}
              </label>
            ))}
          </fieldset>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
