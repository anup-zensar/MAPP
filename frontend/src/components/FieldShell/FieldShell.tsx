import { CircleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { cx } from '@/lib/cx';
import { InfoTip } from '../InfoTip/InfoTip';
import styles from './FieldShell.module.css';

/** A field's marking, shown as a word next to its label (design system §6.2). */
export type FieldMarking = 'Required' | 'Needed to publish' | 'Needed to submit';

/** The parts of the frame every NorthStar field has around its control. */
export interface FieldFrameProps {
  label: string;
  /** Shown as a word next to the label. */
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
  /** The control; it receives the ids of the texts that describe it, for aria-describedby. */
  children: (describedBy: string | undefined) => ReactNode;
}

/**
 * The frame of a form field (design system §7): label above with the marking as a word and an
 * optional ⓘ, a hint, the control, a note and the problem under it. Screen readers hear the
 * marking, hint, note and problem with the control. Internal to @/components.
 */
export function FieldShell({
  id,
  label,
  marking,
  hint,
  tip,
  error,
  note,
  as = 'label',
  labelHidden = false,
  children,
}: FieldShellProps) {
  const markingId = `${id}-marking`;
  const hintId = `${id}-hint`;
  const noteId = `${id}-note`;
  const errorId = `${id}-error`;
  const isGroup = as === 'fieldset';
  // In a fieldset the legend holds the marking, so only the other texts describe the group.
  const describedBy =
    [!isGroup && marking && markingId, hint && hintId, note && noteId, error && errorId]
      .filter(Boolean)
      .join(' ') || undefined;

  const extras = (
    <>
      {tip && <InfoTip label={label} text={tip} />}
      {marking && !isGroup && (
        <span id={markingId} className={styles.marking}>
          {marking}
        </span>
      )}
    </>
  );
  const rest = (
    <>
      {hint && (
        <p id={hintId} className={cx(styles.hint, styles.full)}>
          {hint}
        </p>
      )}
      <div className={styles.full}>{children(describedBy)}</div>
      {note && (
        <div id={noteId} className={cx(styles.note, styles.full)}>
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
          {marking && ' '}
          {marking && <span className={styles.marking}>{marking}</span>}
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
