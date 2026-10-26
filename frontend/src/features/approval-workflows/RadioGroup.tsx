import { CircleAlert } from 'lucide-react';
import styles from './RadioGroup.module.css';

interface RadioGroupProps<T extends string> {
  legend: string;
  /** The group's name and id, so the form's problem summary can bring the focus here. */
  name: string;
  options: { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
  error?: string;
}

/** A small set of choices shown at once, named by its legend; its problem under it. */
export function RadioGroup<T extends string>({
  legend,
  name,
  options,
  value,
  onChange,
  error,
}: RadioGroupProps<T>) {
  const errorId = `${name}-error`;
  return (
    <fieldset
      id={name}
      tabIndex={-1}
      className={styles.radioGroup}
      aria-describedby={error ? errorId : undefined}
    >
      <legend className={styles.legend}>{legend}</legend>
      <div className={styles.radios}>
        {options.map((option) => (
          <label key={option.value} className={styles.radio}>
            <input
              type="radio"
              name={name}
              value={option.value}
              checked={value === option.value}
              onChange={() => onChange(option.value)}
            />
            {option.label}
          </label>
        ))}
      </div>
      {error && (
        <p id={errorId} className={styles.problem}>
          <CircleAlert aria-hidden="true" />
          {error}
        </p>
      )}
    </fieldset>
  );
}
