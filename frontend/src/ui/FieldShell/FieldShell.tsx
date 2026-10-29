import { CircleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { cx } from '../cx';
import { InfoTip } from '../InfoTip/InfoTip';
import styles from './FieldShell.module.css';

/**
 * A field's marking (design system §6.2): "Required" (to save) is a red asterisk after the label,
 * the others are words next to it (038 FR-063, D-358).
 */
export type FieldMarking = 'Required' | 'Needed to publish' | 'Needed to submit';

/** The parts of the frame every MAPP field has around its control. */
export interface FieldFrameProps {
  label: string;
  /** A red asterisk for "Required", otherwise a word next to the label. */
  marking?: FieldMarking;
  /** A short help text under the label. */
  hint?: string;
  /** A plain explanation shown in an ⓘ tip next to the label (Shell attributes' help text). */
  tip?: string;
  /** What is wrong with the value, shown under the field. */
  error?: string;
  /** Keeps the label for screen readers only, for a field in a table cell. */
  labelHidden?: boolean;
}

interface FieldShellProps extends FieldFrameProps {
  /** The control's id: the label points to it and FormErrorSummary links to it. */
  id: string;
  /**
   * 'label' (default) for one control; 'fieldset' for a group of controls (check boxes, a date
   * range), named by a legend.
   */
  as?: 'label' | 'fieldset';
  /** A neutral note under the control: "Rounded to 1,234.57", "12 characters left", a lock reason. */
  note?: ReactNode;
  /** Keeps the note's row when there is no note, so a note that comes and goes moves nothing. */
  reserveNote?: boolean;
  /**
   * The control; it receives the ids of the texts that describe it, for aria-describedby, and
   * `required` for aria-required (a field required to save).
   */
  children: (describedBy: string | undefined, required: true | undefined) => ReactNode;
}

/**
 * The frame of a form field (design system §7): label above with the marking and an optional ⓘ,
 * a hint, the control, a note and the problem under it. Screen readers hear the marking, hint, note
 * and problem with the control. Internal to @/components.
 */
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

/**
 * What follows a label: for a field required to save a red asterisk, hidden from screen readers,
 * which hear "required" from the description `id` instead (038 FR-063); otherwise the word tag.
 */
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
