import { FieldHelpTip } from '@/features/settings';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import type { components } from '@/api/generated/schema';
import { formatWhole } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { AmountPair } from '../AmountPair';
import { isNegative, remainingWhileTyping, withoutSign } from './budget';
import styles from './BudgetCost.module.css';
import { Fact, Facts, PanelSection } from './PanelSection';
import {
  ALLOCATED_TO_INITIATIVES,
  BUDGET,
  BUDGET_AND_COST,
  BUDGET_USED,
  CALCULATED,
  GOALS,
  INQ,
  LEFT_TO_PLAN,
  MQL,
  NO_BUDGET_SET,
  NO_BUDGET_TO_ALLOCATE_FROM,
  NO_PLANNED_COST,
  overAllocatedBy,
  PLANNED_COST,
  REMAINING_CAMPAIGN_BUDGET,
  REMAINING_FOR_INITIATIVES,
  REVENUE,
} from './texts';
import type { RecordLevel } from './texts';

type Money = components['schemas']['Money'];
type BudgetAndCost = components['schemas']['BudgetAndCost'];
type GoalSums = components['schemas']['GoalSums'];

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
  label: string;
  helpKey: string;
  /** The parent's remaining budget as saved (this record's saved budget already taken off). */
  parentRemaining: Money | null;
  savedOwn: Money | null;
  typed: string;
}

/** The parent's remaining budget under the budget field, following what is typed; never blocks. */
export function RemainingBudgetLine({
  label,
  helpKey,
  parentRemaining,
  savedOwn,
  typed,
}: RemainingBudgetLineProps) {
  const shown = remainingWhileTyping(parentRemaining, savedOwn, typed);
  return (
    <div className={styles.line} aria-live="polite">
      <span className={styles.lineLabel}>
        {label}
        <FieldHelpTip helpKey={helpKey} label={label} />
      </span>
      <RemainingAmount value={shown} none={NO_BUDGET_TO_ALLOCATE_FROM} />
    </div>
  );
}

function CalculatedLabel({ label, helpKey }: { label: string; helpKey: string }) {
  return (
    <>
      {label}
      <span className={styles.calculated}>{CALCULATED}</span>
      <FieldHelpTip helpKey={helpKey} label={label} />
    </>
  );
}

export interface BudgetCostSectionProps {
  level: RecordLevel;
  budget: Money | null;
  budgetAndCost: BudgetAndCost;
  /** The parent's remaining budget (an initiative shows its campaign's). */
  parentRemainingBudget: Money | null;
  /** The budget and planned cost in the reader's display currency, beneath when it differs. */
  display?: { budget: Money | null; plannedCost: Money | null };
}

/** Budget and cost in a details panel: budget, what remains, planned cost, used and left. */
export function BudgetCostSection({
  level,
  budget,
  budgetAndCost,
  parentRemainingBudget,
  display,
}: BudgetCostSectionProps) {
  const { money } = useFormatters();
  const { plannedCost, budgetUsedPercent, overBudget, leftToPlan } = budgetAndCost;
  return (
    <PanelSection id="budget" title={BUDGET_AND_COST}>
      <Facts>
        <Fact label={BUDGET} helpKey={`${level}.budget`}>
          {budget && display?.budget ? (
            <AmountPair primary={budget} secondary={display.budget} />
          ) : (
            <RemainingAmount value={budget} none={NO_BUDGET_SET} />
          )}
        </Fact>
        {level === 'campaign' ? (
          <>
            <Fact label={ALLOCATED_TO_INITIATIVES}>{money(budgetAndCost.allocatedToChildren)}</Fact>
            <Fact label={REMAINING_FOR_INITIATIVES}>
              <RemainingAmount value={budgetAndCost.remainingForChildren} none={NO_BUDGET_SET} />
            </Fact>
          </>
        ) : (
          <Fact label={REMAINING_CAMPAIGN_BUDGET} helpKey="initiative.remainingCampaignBudget">
            <RemainingAmount value={parentRemainingBudget} none={NO_BUDGET_TO_ALLOCATE_FROM} />
          </Fact>
        )}
        <Fact label={<CalculatedLabel label={PLANNED_COST} helpKey={`${level}.plannedCost`} />}>
          {plannedCost ? (
            <AmountPair primary={plannedCost} secondary={display?.plannedCost} />
          ) : (
            <span className={styles.stack}>
              <span>—</span>
              <span className={styles.detail}>{NO_PLANNED_COST}</span>
            </span>
          )}
        </Fact>
        <Fact label={<CalculatedLabel label={BUDGET_USED} helpKey={`${level}.budgetUsed`} />}>
          <BudgetUsed percent={budgetUsedPercent} overBudget={overBudget} />
        </Fact>
        <Fact label={<CalculatedLabel label={LEFT_TO_PLAN} helpKey={`${level}.leftToPlan`} />}>
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

/** The budget-used bar with "125%", red above 100 %; `compact` never wraps, for the plan tree. */
export function BudgetUsed({
  percent,
  overBudget,
  compact = false,
  label = BUDGET_USED,
}: {
  percent: number | null;
  overBudget: boolean;
  compact?: boolean;
  label?: string;
}) {
  if (percent === null) {
    return <span>—</span>;
  }
  return (
    <span className={compact ? styles.usedCompact : styles.used}>
      <ProgressBar value={percent} label={label} tone={overBudget ? 'problem' : undefined} />
      <span className={overBudget ? styles.negative : undefined}>{percent}%</span>
    </span>
  );
}

/** The goals of the activities below: INQ, MQL, Revenue; "—" each without a value. */
export function GoalsSection({ level, goals }: { level: RecordLevel; goals: GoalSums }) {
  const { money } = useFormatters();
  return (
    <PanelSection title={<CalculatedLabel label={GOALS} helpKey={`${level}.goals`} />}>
      <Facts>
        <Fact label={INQ}>{goals.inq === null ? '—' : formatWhole(goals.inq)}</Fact>
        <Fact label={MQL}>{goals.mql === null ? '—' : formatWhole(goals.mql)}</Fact>
        <Fact label={REVENUE}>{money(goals.revenue)}</Fact>
      </Facts>
    </PanelSection>
  );
}
