import { ChevronDown } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useId } from 'react';
import { cx } from '@/lib/cx';
import { FILTER_LABELS, MAX_YEARS, TOO_MANY_YEARS, yearsText } from './allWorkText';
import menuStyles from './FilterMenu.module.css';
import styles from './YearRangeMenu.module.css';

export interface YearRangeMenuProps {
  /** The years that have plans, and the current year, newest first. */
  years: number[];
  from: number;
  to: number;
  /** A new range; `to` is never before `from` nor more than five years after it. */
  onChange: (from: number, to: number) => void;
}

/**
 * Years (016 FR-034): the filter button "Years · 2026" or "Years · 2026–2028" opens From and To,
 * two selects drawn like every select. A range of at most five years: choosing a From moves To
 * along when needed, and To offers only the years that keep the range within five
 * ("Choose at most 5 years.").
 */
export function YearRangeMenu({ years, from, to, onChange }: YearRangeMenuProps) {
  const id = useId();
  const label = FILTER_LABELS.years;
  // The chosen years stay offered, also when no plan has them (any more).
  const offered = [...new Set([...years, from, to])].sort((a, b) => a - b);
  const toChoices = offered.filter((year) => year >= from && year - from < MAX_YEARS);

  function chooseFrom(next: number) {
    const nextTo = to < next ? next : Math.min(to, next + MAX_YEARS - 1);
    onChange(next, nextTo);
  }

  return (
    <Popover.Root>
      <Popover.Trigger asChild>
        <button type="button" className={cx(menuStyles.button, menuStyles.inUse)}>
          {label} <span className={menuStyles.summary}>· {yearsText(from, to)}</span>
          <ChevronDown aria-hidden="true" />
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          className={cx(menuStyles.menu, styles.menu)}
          align="start"
          sideOffset={6}
          aria-label={label}
        >
          <fieldset className={styles.fields}>
            <legend className={styles.legend}>Plans of the years</legend>
            <label className={styles.field} htmlFor={`${id}-from`}>
              From
              <select
                id={`${id}-from`}
                className={styles.select}
                value={from}
                onChange={(event) => chooseFrom(Number(event.target.value))}
              >
                {offered.map((year) => (
                  <option key={year} value={year}>
                    {year}
                  </option>
                ))}
              </select>
            </label>
            <label className={styles.field} htmlFor={`${id}-to`}>
              To
              <select
                id={`${id}-to`}
                className={styles.select}
                value={to}
                aria-describedby={`${id}-hint`}
                onChange={(event) => onChange(from, Number(event.target.value))}
              >
                {toChoices.map((year) => (
                  <option key={year} value={year}>
                    {year}
                  </option>
                ))}
              </select>
            </label>
          </fieldset>
          <p id={`${id}-hint`} className={styles.hint}>
            {TOO_MANY_YEARS}
          </p>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
