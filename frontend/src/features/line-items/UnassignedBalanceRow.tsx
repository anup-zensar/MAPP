import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { cx } from '@/ui/cx';
import { FieldHelpTip } from '@/features/settings';
import { toCents } from '@/format/decimal';
import { amountText, centsInput } from './line-item-math';
import type { UnassignedBalance } from './api';
import {
  ALL_ASSIGNED,
  BALANCE_LABEL,
  exceededText,
  leftText,
  METER_LABEL,
  meterText,
  NO_PLANNED_COST,
  ROUNDING_HINT,
} from './line-item-text';
import styles from './UnassignedBalanceRow.module.css';

/** The system row "Unassigned planned cost (Sys_Gen_Balance)": the balance, a meter and the state. */
export function UnassignedBalanceRow({ balance }: { balance: UnassignedBalance }) {
  const exceeded = balance.state === 'exceeded';
  return (
    <div className={styles.row}>
      <div className={styles.head}>
        <span className={styles.label}>
          {BALANCE_LABEL}
          <FieldHelpTip helpKey="activity.unassignedPlannedCost" label="Unassigned planned cost" />
        </span>
        {balance.balance && (
          <span className={cx(styles.amount, exceeded && styles.problem)}>
            {amountText(balance.balance.amount, balance.balance.currency)}
          </span>
        )}
      </div>
      {balance.plannedCost && (
        <div className={styles.meter}>
          <ProgressBar
            label={METER_LABEL}
            value={percent(balance.assigned.amount, balance.plannedCost.amount)}
            tone={exceeded ? 'problem' : undefined}
          />
          <span className={styles.meterText}>
            {meterText(
              centsInput(toCents(balance.assigned.amount)),
              centsInput(toCents(balance.plannedCost.amount)),
              balance.plannedCost.currency,
            )}
          </span>
        </div>
      )}
      <p className={cx(styles.message, exceeded && styles.problem)}>{message(balance)}</p>
      {balance.roundingHint && <p className={styles.hint}>{ROUNDING_HINT}</p>}
    </div>
  );
}

function message(balance: UnassignedBalance): string {
  if (balance.state === 'noPlannedCost' || !balance.balance) {
    return NO_PLANNED_COST;
  }
  const cents = toCents(balance.balance.amount);
  const code = balance.balance.currency;
  switch (balance.state) {
    case 'left':
      return leftText(amountText(cents, code));
    case 'exceeded':
      return exceededText(amountText(-cents, code));
    default:
      return ALL_ASSIGNED;
  }
}

/** The assigned share of the planned cost, in whole percent; null without a planned cost. */
function percent(assigned: string, plannedCost: string): number | null {
  const total = toCents(plannedCost);
  if (total <= 0n) {
    return null;
  }
  return Number((toCents(assigned) * 100n) / total);
}
