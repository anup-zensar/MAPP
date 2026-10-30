import { useState } from 'react';
import { formatAmount, parseMoneyInput } from '@/format/decimal';
import { cx } from '../cx';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import inputs from '../FieldShell/inputs.module.css';

export interface MoneyFieldProps extends FieldFrameProps {
  name: string;
  value: string;
  onChange: (value: string) => void;
  currency: string;
  note?: string;
}

// Accepts "12500", "12,500.00" and "12.5k" and writes it out in full on leaving; more than two
// decimals are rounded half-up and the field says so (D-190). Other text stays as typed.
export function MoneyField({
  name,
  value,
  onChange,
  currency,
  error,
  note,
  ...frame
}: MoneyFieldProps) {
  // The rounding note belongs to the value it was made for; another value hides it.
  const [rounding, setRounding] = useState<{ value: string; note: string } | null>(null);
  const roundingNote = rounding?.value === value ? rounding.note : undefined;

  function formatOnBlur() {
    const parsed = parseMoneyInput(value);
    if (!parsed) {
      return;
    }
    const written = formatAmount(parsed.amount);
    if (parsed.rounded) {
      setRounding({ value: written, note: `Rounded to ${written}` });
    }
    if (written !== value) {
      onChange(written);
    }
  }

  const notes = [roundingNote, note].filter(Boolean).join(' · ') || undefined;

  return (
    <FieldShell id={name} error={error} note={notes} reserveNote {...frame}>
      {(describedBy, required) => (
        <div className={inputs.withUnit}>
          <input
            id={name}
            name={name}
            type="text"
            inputMode="decimal"
            autoComplete="off"
            value={value}
            onChange={(event) => onChange(event.target.value)}
            onBlur={formatOnBlur}
            aria-invalid={error ? true : undefined}
            // Screen readers hear the currency with the field, before its hint and problem.
            aria-describedby={[`${name}-currency`, describedBy].filter(Boolean).join(' ')}
            aria-required={required}
            className={cx(inputs.input, inputs.numeric, error && inputs.invalid)}
          />
          <span id={`${name}-currency`} className={inputs.unit}>
            {currency}
          </span>
        </div>
      )}
    </FieldShell>
  );
}
