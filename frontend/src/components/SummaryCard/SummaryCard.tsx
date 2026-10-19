import type { ReactNode } from 'react';
import { cx } from '@/lib/cx';
import { InfoTip } from '../InfoTip/InfoTip';
import type { StatusTone } from '../StatusChip/StatusChip';
import { Tooltip } from '../Tooltip/Tooltip';
import styles from './SummaryCard.module.css';

export interface SummaryCardProps {
  label: string;
  /** The number or amount, already formatted ("6", "$1.25M"). */
  value: string | number;
  /** A plain explanation of the label, shown in an info tip. */
  tip?: string;
  /** Colours the value, only when the colour means something (a problem in red). */
  tone?: StatusTone;
  /** The complete value when `value` is shortened ("$1,250,000.00" for "$1.25M"). */
  fullValue?: string;
  /**
   * A line under the value (feature 004): "Over-allocated by $12,500.00", "No plan budget set",
   * or a budget-used ProgressBar.
   */
  detail?: ReactNode;
}

/** One of the row of summary cards under a page header (design system §2). */
export function SummaryCard({ label, value, tip, tone, fullValue, detail }: SummaryCardProps) {
  const isShortened = fullValue !== undefined && fullValue !== String(value);

  return (
    <div className={styles.card}>
      <p className={styles.label}>
        {label}
        {tip && <InfoTip label={label} text={tip} />}
      </p>
      <p className={cx(styles.value, tone && styles[tone])}>
        {isShortened ? (
          // A button, so keyboard and screen-reader users can reach the full value too.
          <Tooltip content={fullValue}>
            <button type="button" className={styles.fullValue} aria-label={fullValue}>
              {value}
            </button>
          </Tooltip>
        ) : (
          value
        )}
      </p>
      {detail && <div className={styles.detail}>{detail}</div>}
    </div>
  );
}
