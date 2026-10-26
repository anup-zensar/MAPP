import { cx } from '../cx';
import { formatWhole } from '@/format/decimal';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import inputs from '../FieldShell/inputs.module.css';
import styles from './TextAreaField.module.css';

export interface TextAreaFieldProps extends FieldFrameProps {
  /** The field's name and the text area's id, so FormErrorSummary can link to it. */
  name: string;
  value: string;
  onChange: (value: string) => void;
  /** The longest text the API takes; the field counts down in its last 50 characters. */
  maxLength?: number;
  /** Visible lines (4 when left out). */
  rows?: number;
}

/** How close to the limit the countdown appears. */
const COUNTDOWN_FROM = 50;

/**
 * A multi-line text field with TextField's layout: descriptions, reasons, help texts. Near its
 * limit it says how many characters are left; past it, how many are too many — typing is never
 * cut off, the save names the problem (constitution III).
 */
export function TextAreaField({
  name,
  value,
  onChange,
  maxLength,
  rows = 4,
  error,
  ...frame
}: TextAreaFieldProps) {
  const left = maxLength === undefined ? undefined : maxLength - [...value].length;
  const countdown =
    left === undefined || left > COUNTDOWN_FROM
      ? undefined
      : left >= 0
        ? `${formatWhole(left)} ${left === 1 ? 'character' : 'characters'} left`
        : `${formatWhole(-left)} ${left === -1 ? 'character' : 'characters'} too many`;

  return (
    <FieldShell id={name} error={error} note={countdown} {...frame}>
      {(describedBy) => (
        <textarea
          id={name}
          name={name}
          rows={rows}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          className={cx(inputs.input, styles.textarea, error && inputs.invalid)}
        />
      )}
    </FieldShell>
  );
}
