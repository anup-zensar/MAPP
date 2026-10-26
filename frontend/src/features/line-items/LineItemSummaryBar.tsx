import type { components } from '@/api/generated/schema';
import { cx } from '@/ui/cx';
import { toCents } from '@/format/decimal';
import styles from './LineItemForm.module.css';
import { amountText, moneyText } from './lineItemMath';
import { SUMMARY_NO_COST } from './lineItemText';

type UnassignedBalance = components['schemas']['UnassignedBalance'];

export interface LineItemSummaryBarProps {
  balance: UnassignedBalance;
  /** What saving changes in the assigned amount, in cents of the balance's currency. */
  change: bigint;
  /** A strip above the form (below 1280 px), or a card in the form's rail (design system §2.4 C). */
  layout?: 'bar' | 'rail';
}

/**
 * The activity's context above the line-item form (FR-006): "Activity planned cost 120,000.00 USD
 * · Unassigned now 20,000.00 USD · After saving 0.00 USD", red when negative; the no-planned-cost
 * sentence instead while the activity has none (D-144).
 */
export function LineItemSummaryBar({ balance, change, layout = 'bar' }: LineItemSummaryBarProps) {
  const frame = layout === 'rail' ? styles.summaryRail : styles.summaryBar;
  if (!balance.plannedCost || !balance.balance) {
    return (
      <div className={frame}>
        <p>{SUMMARY_NO_COST}</p>
      </div>
    );
  }
  const now = toCents(balance.balance.amount);
  const after = now - change;
  const code = balance.balance.currency;
  return (
    <div className={frame} aria-label="Activity summary" role="group">
      <dl className={styles.summaryList}>
        <div>
          <dt>Activity planned cost</dt>
          <dd>{moneyText(balance.plannedCost)}</dd>
        </div>
        <div>
          <dt>Unassigned now</dt>
          <dd className={cx(now < 0n && styles.negative)}>{amountText(now, code)}</dd>
        </div>
        <div>
          <dt>After saving</dt>
          <dd className={cx(after < 0n && styles.negative)}>{amountText(after, code)}</dd>
        </div>
      </dl>
    </div>
  );
}
