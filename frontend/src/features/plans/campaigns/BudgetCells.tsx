import type { components } from '@/api/generated/schema';
import { formatMoneyText } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { AmountPair } from '../AmountPair';
import { differentCurrency } from '../plan-currency';
import { isNegative, withoutSign } from './budget';
import styles from './BudgetCost.module.css';
import type { Campaign } from './api';
import { overAllocatedBy, overBy, percentOfBudget, remainingText } from './texts';

type Money = components['schemas']['Money'];

/**
 * A tree row's budget (FR-019, UI audit PL-03): the amount in the plan currency with the display
 * currency beneath when it differs (016 FR-013), then "Remaining <amount>" for what its children
 * may still be given, or red "Over by <amount>" (read out as "Over-allocated by <amount>"); "—"
 * without a budget.
 */
export function BudgetCell({
  budget,
  displayBudget,
  remaining,
}: {
  budget: Money | null;
  displayBudget?: Money | null;
  /** What remains for the children, in the plan currency; null when not shown. */
  remaining: Money | null;
}) {
  const { money: symbolForm } = useFormatters();
  if (!budget) {
    return <span>—</span>;
  }
  // Two currencies in the cell: every amount in it uses the code form (design system §4.4).
  const money = (value: Money) =>
    differentCurrency(budget, displayBudget) ? formatMoneyText(value) : symbolForm(value);
  return (
    <span className={styles.cell}>
      <AmountPair primary={budget} secondary={displayBudget} />
      {remaining &&
        (isNegative(remaining) ? (
          // "Over by …" on screen keeps the column narrow; screen readers hear the whole word.
          <span className={styles.negativeDetail}>
            <span aria-hidden="true">{overBy(money(withoutSign(remaining)))}</span>
            <span className="visually-hidden">
              {overAllocatedBy(money(withoutSign(remaining)))}
            </span>
          </span>
        ) : (
          <span className={styles.detail}>{remainingText(money(remaining))}</span>
        ))}
    </span>
  );
}

/** A campaign row's budget: its own, and what remains for its initiatives. */
export function CampaignBudgetCell({ campaign }: { campaign: Campaign }) {
  return (
    <BudgetCell
      budget={campaign.budget}
      displayBudget={campaign.display.budget}
      remaining={campaign.budgetAndCost.remainingForChildren}
    />
  );
}

/**
 * A tree row's planned cost (FR-011, FR-013) in the plan currency, the display currency beneath
 * when it differs, and how much of the budget it uses ("38% used", red above 100 %) — one
 * column instead of a separate Budget used column, so the tree fits a 1,440 px screen without
 * clipping its last column.
 */
export function PlannedCostCell({
  cost,
  displayCost,
  percent,
  overBudget,
}: {
  cost: Money | null;
  displayCost?: Money | null;
  percent: number | null;
  overBudget: boolean;
}) {
  return (
    <span className={styles.cell}>
      <AmountPair primary={cost} secondary={displayCost} />
      {percent !== null && (
        <span className={overBudget ? styles.negativeDetail : styles.detail}>
          {percentOfBudget(percent)}
        </span>
      )}
    </span>
  );
}
