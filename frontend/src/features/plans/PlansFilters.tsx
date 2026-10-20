import { ChevronDown, SlidersHorizontal } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useId } from 'react';
import type { PlanListFilters } from './listPreferences';
import { PLAN_TYPE_LABELS, PLAN_TYPES } from './planCurrency';
import styles from './PlansFilters.module.css';

export interface PlansFiltersProps {
  filters: PlanListFilters;
  /** Every year with a plan in the team scope, newest first (PlanList.years). */
  years: number[];
  onChange: (next: Partial<PlanListFilters>) => void;
}

/** How many of the menu's filters differ from their defaults: the "(n)" of **Filters (n)**. */
function filtersInUse(filters: PlanListFilters): number {
  return (
    (filters.yearFrom !== null ? 1 : 0) +
    (filters.types.length > 0 ? 1 : 0) +
    (filters.scope !== 'all' ? 1 : 0)
  );
}

/** The years to offer, oldest first: the years with plans plus the chosen ends of the range. */
function yearChoices(years: number[], filters: PlanListFilters): number[] {
  const all = new Set(years);
  if (filters.yearFrom !== null) all.add(filters.yearFrom);
  if (filters.yearTo !== null) all.add(filters.yearTo);
  return [...all].sort((a, b) => a - b);
}

/**
 * **Filters (n)** of the Plans toolbar (design system §5.2, list toolbar): a button that opens the
 * less-used filters — the years, the plan types and whose plans to show — so the toolbar stays one
 * row. Each change applies at once; the chips under the toolbar show and remove them.
 *
 * **Years** (016 FR-033) is All years or a range from–to among the years with plans: choosing a
 * first year alone gives that one year, and the range never runs backwards (the other end
 * follows). **Plan type** takes several; none ticked means every type.
 */
export function PlansFilters({ filters, years, onChange }: PlansFiltersProps) {
  const id = useId();
  const count = filtersInUse(filters);
  const choices = yearChoices(years, filters);
  const allYears = filters.yearFrom === null || filters.yearTo === null;

  function chooseFrom(text: string) {
    if (!text) {
      onChange({ yearFrom: null, yearTo: null });
      return;
    }
    const from = Number(text);
    const to = filters.yearTo === null || filters.yearTo < from ? from : filters.yearTo;
    onChange({ yearFrom: from, yearTo: to });
  }

  function chooseTo(text: string) {
    const to = Number(text);
    const from = filters.yearFrom === null || filters.yearFrom > to ? to : filters.yearFrom;
    onChange({ yearFrom: from, yearTo: to });
  }

  function toggleType(type: (typeof PLAN_TYPES)[number], on: boolean) {
    const chosen = new Set(filters.types);
    if (on) chosen.add(type);
    else chosen.delete(type);
    onChange({ types: PLAN_TYPES.filter((each) => chosen.has(each)) });
  }

  return (
    <Popover.Root>
      <Popover.Trigger className={styles.trigger}>
        <SlidersHorizontal aria-hidden="true" />
        {count > 0 ? `Filters (${count})` : 'Filters'}
        <ChevronDown aria-hidden="true" />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content className={styles.menu} align="start" sideOffset={6} aria-label="Filters">
          <fieldset className={styles.field}>
            <legend className={styles.label}>Years</legend>
            <div className={styles.row}>
              <label className={styles.sublabel} htmlFor={`${id}-from`}>
                From
                <select
                  id={`${id}-from`}
                  className={styles.select}
                  value={allYears ? '' : String(filters.yearFrom)}
                  onChange={(event) => chooseFrom(event.target.value)}
                >
                  <option value="">All years</option>
                  {choices.map((year) => (
                    <option key={year} value={year}>
                      {year}
                    </option>
                  ))}
                </select>
              </label>
              <label className={styles.sublabel} htmlFor={`${id}-to`}>
                To
                <select
                  id={`${id}-to`}
                  className={styles.select}
                  value={allYears ? '' : String(filters.yearTo)}
                  disabled={allYears}
                  onChange={(event) => chooseTo(event.target.value)}
                >
                  {allYears && <option value="">All years</option>}
                  {choices.map((year) => (
                    <option key={year} value={year}>
                      {year}
                    </option>
                  ))}
                </select>
              </label>
            </div>
          </fieldset>
          <fieldset className={styles.field}>
            <legend className={styles.label}>Plan type</legend>
            {PLAN_TYPES.map((type) => (
              <label key={type} className={styles.check}>
                <input
                  type="checkbox"
                  checked={filters.types.includes(type)}
                  onChange={(event) => toggleType(type, event.target.checked)}
                />
                {PLAN_TYPE_LABELS[type]}
              </label>
            ))}
          </fieldset>
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
