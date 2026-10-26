import { useState } from 'react';
import { formatHundredths, percentSplitEvenly, toHundredths } from '@/format/decimal';
import { Button } from '../Button/Button';
import { ChoiceField } from '../ChoiceField/ChoiceField';
import type { ChoiceOption } from '../ChoiceField/ChoiceField';
import { DecimalField } from '../DecimalField/DecimalField';
import { FieldShell } from '../FieldShell/FieldShell';
import type { FieldFrameProps } from '../FieldShell/FieldShell';
import styles from './SplitEditor.module.css';

/** One chosen value and its share of the cost (contract SplitShare). */
export interface SplitShare {
  optionId: string;
  /** A percentage with at most two decimals, as typed ("60", "33.33"). */
  share: string;
}

export interface SplitEditorProps extends FieldFrameProps {
  /** The field's name; the group's id, so FormErrorSummary can link to it. */
  name: string;
  options: ChoiceOption[];
  value: SplitShare[];
  onChange: (value: SplitShare[]) => void;
}

/**
 * A "multiple choice with % split" attribute (feature 002, FR-027, research R10, design system
 * §5): choose the values, then each one's share of the cost. The running total says whether the
 * shares add up to 100.00 %; Split evenly shares it out, and choosing or removing a value splits it
 * evenly again (and says so). Any total can be saved; only 100.00 % counts as filled in.
 */
export function SplitEditor({ name, options, value, onChange, error, ...frame }: SplitEditorProps) {
  const [resplit, setResplit] = useState(false);
  const labelOf = (optionId: string) => {
    const option = options.find((candidate) => candidate.id === optionId);
    if (!option) {
      return optionId;
    }
    return option.active ? option.label : `${option.label} (inactive)`;
  };

  function splitEvenly(optionIds: string[]): SplitShare[] {
    const shares = percentSplitEvenly(optionIds.length);
    return optionIds.map((optionId, index) => ({ optionId, share: shares[index] ?? '0.00' }));
  }

  function choose(optionIds: string[]) {
    onChange(splitEvenly(optionIds));
    setResplit(optionIds.length > 0);
  }

  function changeShare(index: number, share: string) {
    onChange(value.map((part, at) => (at === index ? { ...part, share } : part)));
    setResplit(false);
  }

  const total = value.reduce((sum, part) => sum + (toHundredths(part.share) ?? 0n), 0n);
  const totalText =
    total === 10_000n
      ? `${formatHundredths(total)} % — complete`
      : total < 10_000n
        ? `${formatHundredths(total)} % — ${formatHundredths(10_000n - total)} % left`
        : `${formatHundredths(total)} % — ${formatHundredths(total - 10_000n)} % too much`;

  return (
    <FieldShell {...frame} id={name} as="fieldset" error={error}>
      {() => (
        <div className={styles.editor}>
          <ChoiceField
            label={`Values of ${frame.label}`}
            labelHidden
            name={`${name}-values`}
            multiple
            options={options}
            value={value.map((part) => part.optionId)}
            onChange={choose}
          />
          {value.length > 0 && (
            <>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th scope="col">Value</th>
                    <th scope="col" className={styles.shareColumn}>
                      Share
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {value.map((part, index) => (
                    <tr key={part.optionId}>
                      <td>{labelOf(part.optionId)}</td>
                      <td className={styles.shareColumn}>
                        <DecimalField
                          label={`Share of ${labelOf(part.optionId)}`}
                          labelHidden
                          name={`${name}-share-${index}`}
                          decimals={2}
                          suffix="%"
                          compact
                          value={part.share}
                          onChange={(share) => changeShare(index, share)}
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div className={styles.footer}>
                <p className={total === 10_000n ? styles.complete : styles.incomplete}>
                  {totalText}
                </p>
                <Button
                  size="sm"
                  onClick={() => onChange(splitEvenly(value.map((part) => part.optionId)))}
                >
                  Split evenly
                </Button>
              </div>
            </>
          )}
          <p aria-live="polite" className={styles.note}>
            {resplit ? 'Shares split evenly again.' : ''}
          </p>
        </div>
      )}
    </FieldShell>
  );
}
