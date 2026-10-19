import { cx } from '@/lib/cx';
import { formatAmount, formatWhole, parseDecimalInput } from '@/lib/decimal';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import inputs from '../FieldShell/inputs.module.css';

export interface DecimalFieldProps extends FieldFrameProps {
  /** The field's name and the input's id, so FormErrorSummary can link to it. */
  name: string;
  /** What is typed, as text; read it with parseDecimalInput (or parseWholeInput) before sending it. */
  value: string;
  onChange: (value: string) => void;
  /**
   * The decimals it takes: 0 for whole numbers (INQ goals, number attributes), 2 for percentages
   * (model rates, split shares), 6 for exchange rates.
   */
  decimals: 0 | 2 | 6;
  /** A unit inside the field on the right: "%", "per USD". */
  suffix?: string;
  /** A neutral note under the field. */
  note?: string;
  /** A shorter field, for a table cell (split shares). */
  compact?: boolean;
}

/**
 * A number field (feature 002): whole numbers, percentages with two decimals or rates with six.
 * On leaving the field, whole numbers are written with grouping ("250,000") and percentages with
 * two decimals ("2.00"); rates stay as typed. More decimals than it takes are never rounded: the
 * form's check or the API says so.
 */
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
  const input = (describedBy: string | undefined) => (
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
      className={cx(inputs.input, inputs.numeric, error && inputs.invalid)}
      style={compact ? { maxWidth: '140px' } : undefined}
    />
  );

  return (
    <FieldShell id={name} error={error} note={note} {...frame}>
      {(describedBy) =>
        suffix ? (
          <div className={inputs.withUnit} style={compact ? { maxWidth: '140px' } : undefined}>
            {input(describedBy)}
            <span id={unitId} className={inputs.unit}>
              {suffix}
            </span>
          </div>
        ) : (
          input(describedBy)
        )
      }
    </FieldShell>
  );
}
