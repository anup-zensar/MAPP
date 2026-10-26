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
  /** Names the pair: "Dates". */
  label: string;
  /**
   * The inputs' ids are `<idPrefix>-startDate` and `<idPrefix>-endDate` ("campaign-startDate",
   * "field-startDate"), so FormErrorSummary can link to each.
   */
  idPrefix: string;
  start: string;
  end: string;
  onChange: (range: DateRange) => void;
  /** The dates the record must stay inside (its parent's); sets the inputs' min and max. */
  allowed?: { start: string; end: string } | null;
  /** Says what the allowed range is: "Must be inside the campaign's dates (01 Feb – 30 Nov 2026)". */
  allowedHint?: string;
  /** The latest end date when there is no allowed range (activities: 31 Dec, 10 years on). */
  maxEnd?: string;
  hint?: string;
  marking?: FieldMarking;
  errors?: { start?: string; end?: string };
  /** A warning that does not block saving (R4: "This campaign runs outside 2026, the plan's year"). */
  warning?: string;
  /** Locks the end date and says why (an ongoing activity has none). */
  endDisabledReason?: string;
}

/**
 * Start and end dates as one field (feature 005, design system §5 "Date range picker"): two
 * native date inputs in a group, kept inside the allowed range, each with its own problem, and the
 * allowed range and any warning under both.
 */
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
      {() => (
        <div className={styles.pair}>
          <DateInput
            id={startId}
            label="Start date"
            value={start}
            min={allowed?.start}
            max={end || allowed?.end}
            error={errors.start}
            onChange={(value) => onChange({ start: value, end })}
          />
          <DateInput
            id={endId}
            label="End date"
            value={end}
            min={start || allowed?.start}
            max={allowed?.end ?? maxEnd}
            error={errors.end}
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
  onChange: (value: string) => void;
}

function DateInput({ id, label, value, min, max, error, disabled, onChange }: DateInputProps) {
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
