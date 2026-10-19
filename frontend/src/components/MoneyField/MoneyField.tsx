import { useState } from 'react';
import { cx } from '@/lib/cx';
import { formatAmount, parseMoneyInput } from '@/lib/decimal';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import inputs from '../FieldShell/inputs.module.css';

export interface MoneyFieldProps extends FieldFrameProps {
  /** The field's name and the input's id, so FormErrorSummary can link to it. */
  name: string;
  /** What is typed, as text ("12,500.00"); read it with parseMoneyInput before sending it. */
  value: string;
  onChange: (value: string) => void;
  /** The ISO code shown inside the field ("USD"). */
  currency: string;
  /** A neutral note under the field, such as where a pre-filled amount came from. */
  note?: string;
}

/**
 * An amount with its currency inside the field (design system §5 "Money input"). It accepts
 * "12500", "12,500.00" and "12.5k", and on leaving the field writes the amount out in full
 * ("12,500.00"). More than two decimals are rounded half-up to the cent, and the field says so
 * ("Rounded to 1,234.57", D-190). Text that is not an amount stays as typed: the form's check or
 * the API says what is wrong.
 *
 * To send it: `parseMoneyInput(value)?.amount` with the currency, as a MoneyInput.
 */
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
    <FieldShell id={name} error={error} note={notes} {...frame}>
      {(describedBy) => (
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
