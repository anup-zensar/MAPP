import { InfoTip, ProgressBar } from '@/components';
import type { components } from '@/api/schema';
import { formatWhole } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import { isNegative, remainingWhileTyping, withoutSign } from './budget';
import styles from './BudgetCost.module.css';
import type { Campaign } from './campaignApi';
import { Fact, Facts, PanelSection } from './PanelSection';
import {
  ALLOCATED_TO_INITIATIVES,
  BUDGET,
  BUDGET_AND_COST,
  BUDGET_USED,
  BUDGET_USED_TIP,
  CALCULATED,
  GOALS,
  GOALS_TIP,
  INQ,
  LEFT_TO_PLAN,
  leftText,
  LEFT_TO_PLAN_TIP,
  MQL,
  NO_BUDGET_SET,
  NO_BUDGET_TO_ALLOCATE_FROM,
  NO_PLANNED_COST,
  overAllocatedBy,
  PLANNED_COST,
  PLANNED_COST_TIP,
  REMAINING_CAMPAIGN_BUDGET,
  REMAINING_FOR_INITIATIVES,
  REVENUE,
} from './texts';
import type { RecordLevel } from './texts';

type Money = components['schemas']['Money'];
type BudgetAndCost = components['schemas']['BudgetAndCost'];
type GoalSums = components['schemas']['GoalSums'];

/** An amount, red with "Over-allocated by …" beneath when below zero; "—" with `none` without one. */
export function RemainingAmount({ value, none }: { value: Money | null; none: string }) {
  const { money } = useFormatters();
  if (!value) {
    return (
      <span className={styles.stack}>
        <span>—</span>
        <span className={styles.detail}>{none}</span>
      </span>
    );
  }
  if (isNegative(value)) {
    return (
      <span className={styles.stack}>
        <span className={styles.negative}>{money(value)}</span>
        <span className={styles.negativeDetail}>{overAllocatedBy(money(withoutSign(value)))}</span>
      </span>
    );
  }
  return <span>{money(value)}</span>;
}

export interface RemainingBudgetLineProps {
  /** "Remaining plan budget" / "Remaining campaign budget". */
  label: string;
  tip: string;
  /** The parent's remaining budget as saved (this record's saved budget already taken off). */
  parentRemaining: Money | null;
  /** This record's budget as saved; null on a create form. */
  savedOwn: Money | null;
  /** What the budget field holds now. */
  typed: string;
}

/**
 * The parent's remaining budget under the budget field, following what is typed (FR-010):
 * red with "Over-allocated by …" below zero (never blocks Save, D-015); "—" with "No budget to
 * allocate from" when the parent has no budget.
 */
export function RemainingBudgetLine({
  label,
  tip,
  parentRemaining,
  savedOwn,
  typed,
}: RemainingBudgetLineProps) {
  const shown = remainingWhileTyping(parentRemaining, savedOwn, typed);
  return (
    <div className={styles.line} aria-live="polite">
      <span className={styles.lineLabel}>
        {label}
        <InfoTip label={label} text={tip} />
      </span>
      <RemainingAmount value={shown} none={NO_BUDGET_TO_ALLOCATE_FROM} />
    </div>
  );
}

/** "Calculated" and the ⓘ formula next to a derived value's label (FR-011). */
function CalculatedLabel({ label, tip }: { label: string; tip: string }) {
  return (
    <>
      {label}
      <span className={styles.calculated}>{CALCULATED}</span>
      <InfoTip label={label} text={tip} />
    </>
  );
}

export interface BudgetCostSectionProps {
  level: RecordLevel;
  budget: Money | null;
  budgetAndCost: BudgetAndCost;
  /** The parent's remaining budget (an initiative shows its campaign's). */
  parentRemainingBudget: Money | null;
}

/**
 * Budget and cost in a details panel (FR-010, FR-011): the budget, what the campaign gave its
 * initiatives and what remains, the remaining campaign budget of an initiative, the planned cost
 * rolled up from below, budget used and what is left to plan.
 */
export function BudgetCostSection({
  level,
  budget,
  budgetAndCost,
  parentRemainingBudget,
}: BudgetCostSectionProps) {
  const { money } = useFormatters();
  const { plannedCost, budgetUsedPercent, overBudget, leftToPlan } = budgetAndCost;
  return (
    <PanelSection id="budget" title={BUDGET_AND_COST}>
      <Facts>
        <Fact label={BUDGET}>
          <RemainingAmount value={budget} none={NO_BUDGET_SET} />
        </Fact>
        {level === 'campaign' ? (
          <>
            <Fact label={ALLOCATED_TO_INITIATIVES}>{money(budgetAndCost.allocatedToChildren)}</Fact>
            <Fact label={REMAINING_FOR_INITIATIVES}>
              <RemainingAmount value={budgetAndCost.remainingForChildren} none={NO_BUDGET_SET} />
            </Fact>
          </>
        ) : (
          <Fact label={REMAINING_CAMPAIGN_BUDGET}>
            <RemainingAmount value={parentRemainingBudget} none={NO_BUDGET_TO_ALLOCATE_FROM} />
          </Fact>
        )}
        <Fact label={<CalculatedLabel label={PLANNED_COST} tip={PLANNED_COST_TIP[level]} />}>
          {plannedCost ? (
            money(plannedCost)
          ) : (
            <span className={styles.stack}>
              <span>—</span>
              <span className={styles.detail}>{NO_PLANNED_COST}</span>
            </span>
          )}
        </Fact>
        <Fact label={<CalculatedLabel label={BUDGET_USED} tip={BUDGET_USED_TIP} />}>
          <BudgetUsed percent={budgetUsedPercent} overBudget={overBudget} />
        </Fact>
        <Fact label={<CalculatedLabel label={LEFT_TO_PLAN} tip={LEFT_TO_PLAN_TIP} />}>
          {leftToPlan ? (
            <span className={isNegative(leftToPlan) ? styles.negative : undefined}>
              {money(leftToPlan)}
            </span>
          ) : (
            '—'
          )}
        </Fact>
      </Facts>
    </PanelSection>
  );
}

/** The budget-used bar with "125 %", red above 100 % (FR-011); "—" without a value. */
export function BudgetUsed({
  percent,
  overBudget,
}: {
  percent: number | null;
  overBudget: boolean;
}) {
  if (percent === null) {
    return <span>—</span>;
  }
  return (
    <span className={styles.used}>
      <ProgressBar value={percent} label={BUDGET_USED} tone={overBudget ? 'problem' : undefined} />
      <span className={overBudget ? styles.negative : undefined}>{percent} %</span>
    </span>
  );
}

/** The goals of the activities below (FR-016): INQ, MQL, Revenue; "—" each without a value. */
export function GoalsSection({ goals }: { goals: GoalSums }) {
  const { money } = useFormatters();
  return (
    <PanelSection
      title={
        <span className={styles.lineLabel}>
          {GOALS}
          <span className={styles.calculated}>{CALCULATED}</span>
          <InfoTip label={GOALS} text={GOALS_TIP} />
        </span>
      }
    >
      <Facts>
        <Fact label={INQ}>{goals.inq === null ? '—' : formatWhole(goals.inq)}</Fact>
        <Fact label={MQL}>{goals.mql === null ? '—' : formatWhole(goals.mql)}</Fact>
        <Fact label={REVENUE}>{money(goals.revenue)}</Fact>
      </Facts>
    </PanelSection>
  );
}

/**
 * A campaign row's budget with what remains for its initiatives beneath (FR-019): "<amount> left",
 * or red "Over-allocated by <amount>"; nothing beneath without a budget.
 */
export function CampaignBudgetCell({ campaign }: { campaign: Campaign }) {
  const { money } = useFormatters();
  // Amounts in the reader's display currency (D-220).
  const { budget } = campaign.display;
  const remaining = campaign.display.budgetAndCost.remainingForChildren;
  if (!budget) {
    return <span>—</span>;
  }
  return (
    <span className={styles.cell}>
      <span>{money(budget)}</span>
      {remaining &&
        (isNegative(remaining) ? (
          <span className={styles.negativeDetail}>
            {overAllocatedBy(money(withoutSign(remaining)))}
          </span>
        ) : (
          <span className={styles.detail}>{leftText(money(remaining))}</span>
        ))}
    </span>
  );
}
