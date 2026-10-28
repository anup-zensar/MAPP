import type { components } from '@/api/generated/schema';
import { cx } from '@/ui/cx';
import { FieldHelpTip } from '@/features/settings';
import { toCents } from '@/format/decimal';
import styles from './LineItemForm.module.css';
import { amountText, moneyText } from './line-item-math';
import { SUMMARY_NO_COST } from './line-item-text';

type UnassignedBalance = components['schemas']['UnassignedBalance'];

export interface LineItemSummaryBarProps {
  balance: UnassignedBalance;
  /** What saving changes in the assigned amount, in cents of the balance's currency. */
  change: bigint;
  layout?: 'bar' | 'rail';
}

/** "Activity planned cost · Unassigned now · After saving" above the line-item form, red below zero. */
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
          <dt>
            Activity planned cost
            <FieldHelpTip helpKey="activity.plannedCost" label="Activity planned cost" />
          </dt>
          <dd>{moneyText(balance.plannedCost)}</dd>
        </div>
        <div>
          <dt>
            Unassigned now
            <FieldHelpTip helpKey="activity.unassignedPlannedCost" label="Unassigned now" />
          </dt>
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
