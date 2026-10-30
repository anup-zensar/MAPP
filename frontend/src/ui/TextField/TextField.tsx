import { CircleAlert, Eye, EyeOff } from 'lucide-react';
import { useState } from 'react';
import { cx } from '../cx';
import { Marking } from '../FieldShell/FieldShell';
import type { FieldMarking } from '../FieldShell/FieldShell';
import { InfoTip } from '../InfoTip/InfoTip';
import { Tooltip } from '../Tooltip/Tooltip';
import styles from './TextField.module.css';

export interface TextFieldProps {
  label: string;
  name: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  hint?: string;
  /** A red asterisk for "Required", otherwise a word next to the label (design system §6.2). */
  marking?: FieldMarking;
  maxLength?: number;
  type?: 'text' | 'email' | 'password';
  autoComplete?: string;
  tip?: string;
  /** Values the browser offers while typing, which can still be ignored (a <datalist>). */
  suggestions?: string[];
}

// Screen readers hear the marking, the hint and the problem with the field. A password field also
// gets the eye button that shows what was typed (use PasswordField).
export function TextField({
  label,
  name,
  value,
  onChange,
  error,
  hint,
  marking,
  maxLength,
  type = 'text',
  autoComplete,
  tip,
  suggestions,
}: TextFieldProps) {
  const [showPassword, setShowPassword] = useState(false);
  const isPassword = type === 'password';
  const markingId = `${name}-marking`;
  const hintId = `${name}-hint`;
  const errorId = `${name}-error`;
  const suggestionsId = `${name}-suggestions`;
  const describedBy = [marking && markingId, hint && hintId, error && errorId]
    .filter(Boolean)
    .join(' ');

  return (
    <div className={styles.field}>
      <div className={styles.labelRow}>
        <label htmlFor={name} className={styles.label}>
          {label}
        </label>
        <Marking id={markingId} marking={marking} />
        {tip && <InfoTip label={label} text={tip} />}
      </div>
      {hint && (
        <p id={hintId} className={styles.hint}>
          {hint}
        </p>
      )}
      <div className={styles.control}>
        <input
          id={name}
          name={name}
          type={isPassword && showPassword ? 'text' : type}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          maxLength={maxLength}
          autoComplete={autoComplete}
          list={suggestions ? suggestionsId : undefined}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy || undefined}
          aria-required={marking === 'Required' || undefined}
          className={cx(styles.input, isPassword && styles.withEye, error && styles.invalid)}
        />
        {isPassword && (
          <ShowPasswordButton
            pressed={showPassword}
            onToggle={() => setShowPassword((shown) => !shown)}
          />
        )}
        {suggestions && (
          <datalist id={suggestionsId}>
            {suggestions.map((suggestion) => (
              <option key={suggestion} value={suggestion} />
            ))}
          </datalist>
        )}
      </div>
      {error && (
        <p id={errorId} className={styles.error}>
          <CircleAlert />
          {error}
        </p>
      )}
    </div>
  );
}

// The eye button's name stays "Show password"; whether the password is shown is told by
// aria-pressed.
function ShowPasswordButton({ pressed, onToggle }: { pressed: boolean; onToggle: () => void }) {
  return (
    <Tooltip content="Show password">
      <button
        type="button"
        className={styles.eye}
        aria-label="Show password"
        aria-pressed={pressed}
        onClick={onToggle}
      >
        {pressed ? <EyeOff /> : <Eye />}
      </button>
    </Tooltip>
  );
}
