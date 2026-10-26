import { CircleAlert } from 'lucide-react';
import type { FieldBinding } from '@/ui/FormField/FormField';
import styles from './rates.module.css';

export interface RateDateFieldProps extends FieldBinding<string> {
  /** Under a date after today: from when the rate counts. */
  note?: string;
}

/** The date of a new rate: today unless changed. */
export function RateDateField({ name, value, onChange, error, note }: RateDateFieldProps) {
  return (
    <div className={styles.field}>
      <label htmlFor="rate-date" className={styles.label}>
        Date<span className={styles.marking}>Required</span>
      </label>
      <input
        id="rate-date"
        name={name}
        type="date"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        aria-invalid={error ? true : undefined}
        aria-describedby={
          [note && 'rate-date-note', error && 'rate-date-error'].filter(Boolean).join(' ') ||
          undefined
        }
        className={`${styles.input} ${error ? styles.invalid : ''}`}
      />
      {note && (
        <p id="rate-date-note" className={styles.note}>
          {note}
        </p>
      )}
      {error && (
        <p id="rate-date-error" className={styles.error}>
          <CircleAlert aria-hidden="true" />
          {error}
        </p>
      )}
    </div>
  );
}
