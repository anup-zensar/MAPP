import { CircleAlert } from 'lucide-react';
import { useId } from 'react';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import type { OverspendMode } from './api';
import type { BudgetCheck } from './overspend-check';
import { MODE_HINTS, MODE_WORDS, MODES, OVERSPEND_LABEL } from './overspend-text';
import styles from './OverspendField.module.css';

export interface OverspendFieldProps {
  name: string;
  value: OverspendMode;
  onChange: (mode: OverspendMode) => void;
  /** Why the person may not change it; empty when they may. */
  lockedReason?: string;
  check?: BudgetCheck;
}

/** "When planned cost goes over this budget": Warn or Refuse, with what saving would do (037 FR-002). */
export function OverspendField({
  name,
  value,
  onChange,
  lockedReason,
  check,
}: OverspendFieldProps) {
  const id = useId();
  const locked = Boolean(lockedReason);
  const described = [locked ? `${id}-lock` : '', check?.error ? `${id}-error` : '']
    .filter(Boolean)
    .join(' ');
  return (
    <fieldset id={name} className={styles.group} aria-describedby={described || undefined}>
      <legend className={styles.legend}>{OVERSPEND_LABEL}</legend>
      {MODES.map((mode) => (
        <label key={mode} className={styles.radio}>
          <input
            type="radio"
            name={name}
            value={mode}
            checked={value === mode}
            disabled={locked}
            onChange={() => onChange(mode)}
          />
          <span>
            <span className={styles.word}>{MODE_WORDS[mode]}</span>
            <span className={styles.hint}>{MODE_HINTS[mode]}</span>
          </span>
        </label>
      ))}
      {locked && <LockedNote id={`${id}-lock`}>{lockedReason}</LockedNote>}
      {check?.note && <p className={styles.note}>{check.note}</p>}
      {check?.error && (
        <p id={`${id}-error`} className={styles.error}>
          <CircleAlert aria-hidden="true" />
          {check.error}
        </p>
      )}
    </fieldset>
  );
}
