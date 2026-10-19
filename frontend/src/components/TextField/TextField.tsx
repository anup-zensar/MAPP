import { CircleAlert, Eye, EyeOff } from 'lucide-react';
import { useState } from 'react';
import { cx } from '@/lib/cx';
import { Tooltip } from '../Tooltip/Tooltip';
import styles from './TextField.module.css';

export interface TextFieldProps {
  label: string;
  /** The field's name. It is also the input's id, so FormErrorSummary can link to the field. */
  name: string;
  value: string;
  onChange: (value: string) => void;
  /** What is wrong with the value, shown under the field. */
  error?: string;
  /** A short help text under the label. */
  hint?: string;
  /** Shown as a word next to the label (design system §6.2). */
  marking?: 'Required' | 'Needed to publish' | 'Needed to submit';
  maxLength?: number;
  type?: 'text' | 'email' | 'password';
  /** The browser's autofill hint, for example "username" or "current-password". */
  autoComplete?: string;
}

/**
 * A labelled text input (design system §7): the label above, the marking as a word, the problem
 * under the field. Screen readers hear the marking, the hint and the problem with the field.
 * A password field also gets the eye button that shows what was typed (use PasswordField).
 */
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
}: TextFieldProps) {
  const [showPassword, setShowPassword] = useState(false);
  const isPassword = type === 'password';
  const markingId = `${name}-marking`;
  const hintId = `${name}-hint`;
  const errorId = `${name}-error`;
  const describedBy = [marking && markingId, hint && hintId, error && errorId]
    .filter(Boolean)
    .join(' ');

  return (
    <div className={styles.field}>
      <div className={styles.labelRow}>
        <label htmlFor={name} className={styles.label}>
          {label}
        </label>
        {marking && (
          <span id={markingId} className={styles.marking}>
            {marking}
          </span>
        )}
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
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy || undefined}
          className={cx(styles.input, isPassword && styles.withEye, error && styles.invalid)}
        />
        {isPassword && (
          <ShowPasswordButton
            pressed={showPassword}
            onToggle={() => setShowPassword((shown) => !shown)}
          />
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

/**
 * The eye button of a password field. Its name stays "Show password"; whether the password is
 * shown is told by aria-pressed (research R17).
 */
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
