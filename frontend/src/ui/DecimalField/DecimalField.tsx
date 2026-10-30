import { formatAmount, formatWhole, parseDecimalInput } from '@/format/decimal';
import { cx } from '../cx';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import inputs from '../FieldShell/inputs.module.css';

export interface DecimalFieldProps extends FieldFrameProps {
  name: string;
  /** What is typed, as text; read it with parseDecimalInput (or parseWholeInput) before sending it. */
  value: string;
  onChange: (value: string) => void;
  decimals: 0 | 2 | 6;
  suffix?: string;
  note?: string;
  compact?: boolean;
}

// On leaving the field whole numbers get grouping ("250,000") and percentages two decimals ("2.00");
// rates stay as typed. Extra decimals are never rounded: the form's check or the API says so.
export function DecimalField({
  name,
  value,
  onChange,
  decimals,
  suffix,
  error,
  note,
  compact = false,
  ...frame
}: DecimalFieldProps) {
  function formatOnBlur() {
    const parsed = parseDecimalInput(value, decimals);
    if (parsed === null || decimals === 6) {
      return;
    }
    const written = decimals === 0 ? formatWhole(parsed) : formatAmount(parsed);
    if (written !== value) {
      onChange(written);
    }
  }

  const unitId = `${name}-unit`;
  const input = (describedBy: string | undefined, required: true | undefined) => (
    <input
      id={name}
      name={name}
      type="text"
      inputMode={decimals === 0 ? 'numeric' : 'decimal'}
      autoComplete="off"
      value={value}
      onChange={(event) => onChange(event.target.value)}
      onBlur={formatOnBlur}
      aria-invalid={error ? true : undefined}
      aria-describedby={[suffix && unitId, describedBy].filter(Boolean).join(' ') || undefined}
      aria-required={required}
      className={cx(inputs.input, inputs.numeric, error && inputs.invalid)}
      style={compact ? { maxWidth: '140px' } : undefined}
    />
  );

  return (
    <FieldShell id={name} error={error} note={note} {...frame}>
      {(describedBy, required) =>
        suffix ? (
          <div className={inputs.withUnit} style={compact ? { maxWidth: '140px' } : undefined}>
            {input(describedBy, required)}
            <span id={unitId} className={inputs.unit}>
              {suffix}
            </span>
          </div>
        ) : (
          input(describedBy, required)
        )
      }
    </FieldShell>
  );
}
