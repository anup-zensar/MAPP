import { cx } from '@/ui/cx';
import { FieldShell } from '@/ui/FieldShell/FieldShell';
import type { FieldFrameProps } from '@/ui/FieldShell/FieldShell';
import inputs from '@/ui/FieldShell/inputs.module.css';

export interface DateFieldProps extends FieldFrameProps {
  name: string;
  /** "YYYY-MM-DD", or "" when empty. */
  value: string;
  onChange: (value: string) => void;
}

/** A calendar date in the browser's date input, in the field frame of every MAPP field. */
export function DateField({ name, value, onChange, error, ...frame }: DateFieldProps) {
  return (
    <FieldShell id={name} error={error} {...frame}>
      {(describedBy) => (
        <input
          id={name}
          name={name}
          type="date"
          value={value}
          onChange={(event) => onChange(event.target.value)}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          className={cx(inputs.input, error && inputs.invalid)}
        />
      )}
    </FieldShell>
  );
}
