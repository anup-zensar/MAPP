import { CircleAlert } from 'lucide-react';
import { InfoTip } from '../InfoTip/InfoTip';
import { LockedNote } from '../LockedNote/LockedNote';
import styles from './CheckboxField.module.css';

export interface CheckboxFieldProps {
  label: string;
  /** The field's name and the check box's id, so FormErrorSummary can link to it. */
  name: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  /** A short help text under the label. */
  hint?: string;
  /** A plain explanation in an ⓘ tip next to the label. */
  tip?: string;
  error?: string;
  /** Locks the check box and says why under it ("Ongoing can't be undone after saving."). */
  disabledReason?: string;
}

/**
 * A labelled tick box (design system §7): yes/no Shell attributes, "Strategic / AOP plan",
 * "Ongoing (no end date)". A locked one keeps its value and says why.
 */
export function CheckboxField({
  label,
  name,
  checked,
  onChange,
  hint,
  tip,
  error,
  disabledReason,
}: CheckboxFieldProps) {
  const hintId = `${name}-hint`;
  const reasonId = `${name}-reason`;
  const errorId = `${name}-error`;
  const describedBy =
    [hint && hintId, disabledReason && reasonId, error && errorId].filter(Boolean).join(' ') ||
    undefined;

  return (
    <div className={styles.field}>
      <div className={styles.row}>
        <input
          id={name}
          name={name}
          type="checkbox"
          checked={checked}
          disabled={Boolean(disabledReason)}
          onChange={(event) => onChange(event.target.checked)}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          className={styles.box}
        />
        <label htmlFor={name} className={styles.label}>
          {label}
        </label>
        {tip && <InfoTip label={label} text={tip} />}
      </div>
      {hint && (
        <p id={hintId} className={styles.hint}>
          {hint}
        </p>
      )}
      {disabledReason && (
        <div className={styles.indented}>
          <LockedNote id={reasonId}>{disabledReason}</LockedNote>
        </div>
      )}
      {error && (
        <p id={errorId} className={styles.error}>
          <CircleAlert />
          {error}
        </p>
      )}
    </div>
  );
}
