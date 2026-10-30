import { CircleAlert, TriangleAlert } from 'lucide-react';
import { cx } from '../cx';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldMarking } from '../FieldShell/FieldShell';
import inputs from '../FieldShell/inputs.module.css';
import styles from './DateRangeField.module.css';

/** A start and an end date, "YYYY-MM-DD" or "" when not set. */
export interface DateRange {
  start: string;
  end: string;
}

export interface DateRangeFieldProps {
  label: string;
  /** The inputs are `<idPrefix>-startDate` and `<idPrefix>-endDate`, for FormErrorSummary's links. */
  idPrefix: string;
  start: string;
  end: string;
  onChange: (range: DateRange) => void;
  /** The dates the record must stay inside (its parent's); sets the inputs' min and max. */
  allowed?: { start: string; end: string } | null;
  allowedHint?: string;
  /** The latest end date when there is no allowed range (activities: 31 Dec, 10 years on). */
  maxEnd?: string;
  hint?: string;
  tip?: string;
  marking?: FieldMarking;
  errors?: { start?: string; end?: string };
  warning?: string;
  endDisabledReason?: string;
}

/** Start and end dates as one field, kept inside the allowed range, each with its own problem. */
export function DateRangeField({
  label,
  idPrefix,
  start,
  end,
  onChange,
  allowed,
  allowedHint,
  maxEnd,
  hint,
  tip,
  marking,
  errors = {},
  warning,
  endDisabledReason,
}: DateRangeFieldProps) {
  const startId = `${idPrefix}-startDate`;
  const endId = `${idPrefix}-endDate`;
  const notes = [allowedHint, endDisabledReason].filter(Boolean);

  return (
    <FieldShell
      id={`${idPrefix}-dates`}
      as="fieldset"
      label={label}
      marking={marking}
      hint={hint}
      tip={tip}
      note={
        notes.length > 0 || warning ? (
          <>
            {notes.map((note) => (
              <p key={note}>{note}</p>
            ))}
            {warning && (
              <p className={styles.warning}>
                <TriangleAlert aria-hidden="true" />
                {warning}
              </p>
            )}
          </>
        ) : undefined
      }
    >
      {(_, required) => (
        <div className={styles.pair}>
          <DateInput
            id={startId}
            label="Start date"
            value={start}
            min={allowed?.start}
            max={end || allowed?.end}
            error={errors.start}
            required={required}
            onChange={(value) => onChange({ start: value, end })}
          />
          <DateInput
            id={endId}
            label="End date"
            value={end}
            min={start || allowed?.start}
            max={allowed?.end ?? maxEnd}
            error={errors.end}
            required={required}
            disabled={Boolean(endDisabledReason)}
            onChange={(value) => onChange({ start, end: value })}
          />
        </div>
      )}
    </FieldShell>
  );
}

interface DateInputProps {
  id: string;
  label: string;
  value: string;
  min?: string;
  max?: string;
  error?: string;
  disabled?: boolean;
  required?: true;
  onChange: (value: string) => void;
}

function DateInput(props: DateInputProps) {
  const { id, label, value, min, max, error, disabled, required, onChange } = props;
  const errorId = `${id}-error`;
  return (
    <div className={styles.date}>
      <label htmlFor={id} className={styles.label}>
        {label}
      </label>
      <input
        id={id}
        type="date"
        value={value}
        min={min}
        max={max}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errorId : undefined}
        aria-required={required}
        className={cx(inputs.input, error && inputs.invalid)}
      />
      {error && (
        <p id={errorId} className={styles.error}>
          <CircleAlert />
          {error}
        </p>
      )}
    </div>
  );
}
