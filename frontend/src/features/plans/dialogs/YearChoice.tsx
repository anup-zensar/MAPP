import { CircleAlert } from 'lucide-react';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import type { PlanCopyOptions } from '../api';
import styles from './dialogs.module.css';

export interface YearChoiceProps {
  options: PlanCopyOptions;
  year: number;
  error?: string;
  onChange: (year: number) => void;
}

/** The copy's year: the same or the next one, which is unavailable with the reason (FR-028). */
export function YearChoice({ options, year, error, onChange }: YearChoiceProps) {
  const nextYear = options.nextYear;
  return (
    <fieldset className={styles.radioGroup} aria-describedby={error ? 'year-error' : undefined}>
      <legend className={styles.legend}>Year</legend>
      <label className={styles.radio}>
        <input
          type="radio"
          name="year"
          value={options.sameYear}
          checked={year === options.sameYear}
          onChange={() => onChange(options.sameYear)}
        />
        Same year ({options.sameYear})
      </label>
      <div>
        <label className={styles.radio}>
          <input
            type="radio"
            name="year"
            value={nextYear.year}
            checked={year === nextYear.year}
            disabled={!nextYear.available}
            aria-describedby={nextYear.available ? undefined : 'next-year-reason'}
            onChange={() => onChange(nextYear.year)}
          />
          Next year ({nextYear.year})
        </label>
        {!nextYear.available && (
          <div className={styles.radioNote}>
            <LockedNote id="next-year-reason">{nextYear.reason}</LockedNote>
          </div>
        )}
      </div>
      {year === nextYear.year && <p className={styles.note}>Every date moves one year later.</p>}
      {error && (
        <p id="year-error" className={styles.problem}>
          <CircleAlert aria-hidden="true" />
          {error}
        </p>
      )}
    </fieldset>
  );
}
