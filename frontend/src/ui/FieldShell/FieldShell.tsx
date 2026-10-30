import { CircleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { cx } from '../cx';
import { InfoTip } from '../InfoTip/InfoTip';
import styles from './FieldShell.module.css';

/** "Required" (to save) is a red asterisk after the label; the others are words next to it. */
export type FieldMarking = 'Required' | 'Needed to publish' | 'Needed to submit';

export interface FieldFrameProps {
  label: string;
  marking?: FieldMarking;
  hint?: string;
  tip?: string;
  error?: string;
  labelHidden?: boolean;
}

interface FieldShellProps extends FieldFrameProps {
  id: string;
  as?: 'label' | 'fieldset';
  note?: ReactNode;
  /** Keeps the note's row when there is no note, so a note that comes and goes moves nothing. */
  reserveNote?: boolean;
  /** The control, given the ids for aria-describedby and `required` for aria-required. */
  children: (describedBy: string | undefined, required: true | undefined) => ReactNode;
}

/** A form field's frame (design system §7): label, marking, ⓘ, hint, control, note and problem. */
export function FieldShell({
  id,
  label,
  marking,
  hint,
  tip,
  error,
  note,
  reserveNote = false,
  as = 'label',
  labelHidden = false,
  children,
}: FieldShellProps) {
  const markingId = `${id}-marking`;
  const hintId = `${id}-hint`;
  const noteId = `${id}-note`;
  const errorId = `${id}-error`;
  const isGroup = as === 'fieldset';
  const required = marking === 'Required' || undefined;
  // In a fieldset the legend holds a word marking, so only the other texts describe the group.
  const describedBy =
    [
      (!isGroup || required) && marking && markingId,
      hint && hintId,
      note && noteId,
      error && errorId,
    ]
      .filter(Boolean)
      .join(' ') || undefined;

  const extras = (
    <>
      {!isGroup && <Marking id={markingId} marking={marking} />}
      {tip && <InfoTip label={label} text={tip} />}
    </>
  );
  const rest = (
    <>
      {hint && (
        <p id={hintId} className={cx(styles.hint, styles.full)}>
          {hint}
        </p>
      )}
      <div className={styles.full}>{children(describedBy, required)}</div>
      {(note || reserveNote) && (
        <div id={noteId} className={cx(styles.note, styles.full, reserveNote && styles.reserved)}>
          {note}
        </div>
      )}
      {error && (
        <p id={errorId} className={cx(styles.error, styles.full)}>
          <CircleAlert />
          {error}
        </p>
      )}
    </>
  );

  if (isGroup) {
    return (
      <fieldset className={styles.group} aria-describedby={describedBy} id={id} tabIndex={-1}>
        {/* Floated, so the legend lines up with the ⓘ like a label does. */}
        <legend className={cx(styles.legend, labelHidden && 'visually-hidden')}>
          <span className={styles.label}>{label}</span>
          {marking && !required && ' '}
          <Marking id={markingId} marking={marking} />
        </legend>
        {extras}
        {rest}
      </fieldset>
    );
  }

  return (
    <div className={styles.field}>
      <div className={labelHidden ? 'visually-hidden' : styles.labelRow}>
        <label htmlFor={id} className={styles.label}>
          {label}
        </label>
        {extras}
      </div>
      {rest}
    </div>
  );
}

/** The asterisk is hidden from screen readers, which hear "required" from the description. */
export function Marking({ id, marking }: { id: string; marking: FieldMarking | undefined }) {
  if (marking === 'Required') {
    return (
      <>
        <span aria-hidden="true" className={styles.asterisk}>
          *
        </span>
        <span id={id} hidden>
          required
        </span>
      </>
    );
  }
  return marking ? (
    <span id={id} className={styles.marking}>
      {marking}
    </span>
  ) : null;
}
