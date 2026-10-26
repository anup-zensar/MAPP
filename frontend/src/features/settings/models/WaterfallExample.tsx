import { useId, useState } from 'react';
import { DecimalField } from '@/ui/DecimalField/DecimalField';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { formatAmount, formatWhole, parseWholeInput } from '@/lib/decimal';
import { plural } from '@/lib/text';
import { waterfall } from '@/lib/waterfall';
import styles from './WaterfallExample.module.css';

/** The most inquiries the example takes. */
const MAX_INQUIRIES = 1_000_000_000n;

export interface WaterfallExampleProps {
  /** The INQ→MQL rate as a percentage ("2", "2.00"); null when it is not filled in or not a rate. */
  inqToMqlRate: string | null;
  mqlToDealRate: string | null;
  /** The average deal size in USD ("10000.00"); null when it is not filled in or not an amount. */
  averageDealSize: string | null;
}

/**
 * The model's worked example (FR-010, SC-006): for an example number of inquiries (250,000 at
 * first; changing it saves nothing) "250,000 INQ → 5,000 MQL → 250 deals → 2,500,000.00 USD
 * revenue", worked out as goals are (lib/waterfall.ts) and updated as soon as any value changes.
 */
export function WaterfallExample({
  inqToMqlRate,
  mqlToDealRate,
  averageDealSize,
}: WaterfallExampleProps) {
  const titleId = useId();
  const [inquiries, setInquiries] = useState('250,000');
  const inq = parseWholeInput(inquiries);
  const inqValid = inq !== null && inq >= 0n && inq <= MAX_INQUIRIES;

  let result: string | null = null;
  if (inqValid) {
    if (inqToMqlRate === null || mqlToDealRate === null || averageDealSize === null) {
      result = 'Fill in both rates and the average deal size to see the example.';
    } else {
      const goals = waterfall(inq, {
        inqToMqlRate,
        mqlToDealRate,
        averageDealSize: { amount: averageDealSize },
      });
      result =
        goals === 'tooLarge'
          ? 'Too large to calculate — try fewer inquiries.'
          : `${formatWhole(inq)} INQ → ${formatWhole(goals.mql)} MQL → ` +
            `${plural(goals.deals, 'deal')} → ${formatAmount(goals.revenue)} USD revenue`;
    }
  }

  return (
    <section aria-labelledby={titleId} className={styles.example}>
      <div className={styles.header}>
        <h3 id={titleId} className={styles.title}>
          Worked example
        </h3>
        <InfoTip
          label="Worked example"
          text="MQL = inquiries × INQ→MQL rate and deals = MQL × MQL→deal rate, each rounded to a whole number; revenue = deals × average deal size."
        />
        <span className={styles.calculated}>Calculated — not saved</span>
      </div>
      <DecimalField
        label="Example number of inquiries"
        name="exampleInquiries"
        decimals={0}
        value={inquiries}
        onChange={setInquiries}
        error={inqValid ? undefined : 'Enter a whole number from 0 to 1,000,000,000.'}
      />
      <output className={styles.result} aria-labelledby={titleId}>
        {result}
      </output>
    </section>
  );
}
