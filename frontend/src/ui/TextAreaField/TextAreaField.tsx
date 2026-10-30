import { formatWhole } from '@/format/decimal';
import { cx } from '../cx';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import inputs from '../FieldShell/inputs.module.css';
import styles from './TextAreaField.module.css';

export interface TextAreaFieldProps extends FieldFrameProps {
  name: string;
  value: string;
  onChange: (value: string) => void;
  /** The longest text the API takes; the field counts down in its last 50 characters. */
  maxLength?: number;
  rows?: number;
}

const COUNTDOWN_FROM = 50;

// Past its limit it says how many characters are too many: typing is never cut off, the save
// names the problem (constitution III).
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
      {(describedBy, required) => (
        <textarea
          id={name}
          name={name}
          rows={rows}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          aria-required={required}
          className={cx(inputs.input, styles.textarea, error && inputs.invalid)}
        />
      )}
    </FieldShell>
  );
}
