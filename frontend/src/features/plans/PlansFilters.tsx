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
  /**
   * The API filters by a range of years and by plan type (016). Until 016's backend is merged it
   * takes one year only and knows no plan types, so the menu offers one Year and no Plan type.
   */
  served016?: boolean;
  onChange: (next: Partial<PlanListFilters>) => void;
}

const LAYOUTS = [
  ['cards', 'Cards'],
  ['table', 'Table'],
] as const;

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
 * follows). **Plan type** takes several; none ticked means every type. Both need 016's API; before
 * it the menu keeps the single Year it had.
 *
 * On phones (below 600 px, UI audit P-06) the button is its icon with the number of filters in
 * use, so the toolbar keeps two rows, and the menu also holds the Cards / Table layout.
 */
export function PlansFilters({ filters, years, served016 = true, onChange }: PlansFiltersProps) {
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
    const to =
      !served016 || filters.yearTo === null || filters.yearTo < from ? from : filters.yearTo;
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
        <span className={styles.word}>{count > 0 ? `Filters (${count})` : 'Filters'}</span>
        {count > 0 && (
          <span className={styles.badge} aria-hidden="true">
            {count}
          </span>
        )}
        <ChevronDown aria-hidden="true" className={styles.chevron} />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content className={styles.menu} align="start" sideOffset={6} aria-label="Filters">
          {!served016 ? (
            <div className={styles.field}>
              <label htmlFor={`${id}-from`} className={styles.label}>
                Year
              </label>
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
            </div>
          ) : (
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
          )}
          {served016 && (
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
          )}
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
          {/* Phones only: the toolbar has no room for the layout switch. */}
          <fieldset className={`${styles.field} ${styles.phoneOnly}`}>
            <legend className={styles.label}>Layout</legend>
            {LAYOUTS.map(([value, label]) => (
              <label key={value} className={styles.radio}>
                <input
                  type="radio"
                  name={`${id}-layout`}
                  value={value}
                  checked={filters.layout === value}
                  onChange={() => onChange({ layout: value })}
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
