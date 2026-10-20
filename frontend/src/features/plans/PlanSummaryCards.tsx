import type { ReactNode } from 'react';
import { ProgressBar, SummaryCard } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import type { Plan } from './plansApi';
import styles from './PlanSummaryCards.module.css';
import { APPROVAL_PROGRESS } from './planText';

export interface PlanSummaryCardsProps {
  plan: Plan;
  /** Under the plan budget (014: where an allocated budget comes from). */
  budgetNote?: ReactNode;
}

/**
 * The plan's money and progress at a glance (FR-012, design system §2.4 B: four cards): the plan
 * budget with what remains for campaigns (red "Over-allocated by …" below zero), the planned cost,
 * the budget used (red above 100 %) and the activities approved — each with its formula in a tip
 * and "—" with the reason when it can't be shown.
 */
export function PlanSummaryCards({ plan, budgetNote }: PlanSummaryCardsProps) {
  const { money, moneyCompact } = useFormatters();
  const { figures } = plan;
  const noBudget = <span>No plan budget set</span>;

  return (
    // #budget: where Home's "Open budget" leads (feature 009).
    <section id="budget" className={styles.summary} aria-label="Summary">
      <SummaryCard
        label="Plan budget"
        value={moneyCompact(figures.budget)}
        fullValue={money(figures.budget)}
        tip="The money this plan may spend in its year. Remaining for campaigns = plan budget − the budgets given to campaigns; it can go below zero."
        detail={
          <span className={styles.details}>
            {!figures.budget || !figures.remainingBudget ? (
              noBudget
            ) : figures.overAllocatedBy ? (
              <span className={styles.problem}>
                Over-allocated by {money(figures.overAllocatedBy)}
              </span>
            ) : (
              <span>Remaining for campaigns {money(figures.remainingBudget)}</span>
            )}
            {budgetNote}
          </span>
        }
      />
      <SummaryCard
        label="Planned cost"
        value={moneyCompact(figures.plannedCost)}
        fullValue={money(figures.plannedCost)}
        tip="The planned cost of every activity in the plan, added up."
        detail={figures.plannedCost ? undefined : <span>Nothing planned yet</span>}
      />
      <SummaryCard
        label="Budget used"
        value={figures.budgetUsedPercent === null ? '—' : `${figures.budgetUsedPercent}%`}
        tip="Planned cost ÷ plan budget."
        tone={figures.overBudget ? 'problem' : undefined}
        detail={
          figures.budgetUsedPercent !== null ? (
            <ProgressBar
              value={figures.budgetUsedPercent}
              label="Budget used"
              tone={figures.overBudget ? 'problem' : undefined}
            />
          ) : figures.budget ? (
            <span>Plan budget is 0</span>
          ) : (
            noBudget
          )
        }
      />
      <SummaryCard
        label="Activities approved"
        value={`${figures.activitiesApproved} of ${figures.activities}`}
        tip="Activities that are Approved, In progress or Complete, out of all activities in the plan."
        detail={<span>{APPROVAL_PROGRESS[figures.approvalProgress].word}</span>}
      />
    </section>
  );
}
