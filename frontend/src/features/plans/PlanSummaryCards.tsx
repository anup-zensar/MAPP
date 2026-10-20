import { Link } from 'react-router';
import type { components } from '@/api/schema';
import { ProgressBar, SummaryCard } from '@/components';
import { formatMoneyText } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import { differentCurrency } from './planCurrency';
import type { Plan } from './plansApi';
import styles from './PlanSummaryCards.module.css';
import { APPROVAL_PROGRESS } from './planText';

type Money = components['schemas']['Money'];
type PlanAllocation = components['schemas']['PlanAllocation'];

export interface PlanSummaryCardsProps {
  plan: Plan;
}

/**
 * The plan's money and progress at a glance (FR-012, design system §2.4 B: four cards): the plan
 * budget with what remains for campaigns (red "Over-allocated by …" below zero) and, for an
 * allocated plan, where it comes from (014 FR-031); the planned cost; the budget used (red above
 * 100 %); the activities approved — each with its formula in a tip and "—" with the reason when it
 * can't be shown. Amounts are in the plan currency with the display currency beneath when it
 * differs (016 FR-013).
 */
export function PlanSummaryCards({ plan }: PlanSummaryCardsProps) {
  const { money, moneyCompact } = useFormatters();
  const { figures } = plan;
  // The plan currency's figures; the display currency's until 016's API sends them.
  const own = plan.planFigures ?? figures;
  const noBudget = <span>No plan budget set</span>;

  return (
    // #budget: where Home's "Open budget" leads (feature 009).
    <section id="budget" className={styles.summary} aria-label="Summary">
      <SummaryCard
        label="Plan budget"
        value={moneyCompact(own.budget)}
        fullValue={money(own.budget)}
        tip="The money this plan may spend in its year. Remaining for campaigns = plan budget − the budgets given to campaigns; it can go below zero."
        detail={
          <span className={styles.details}>
            <Beside own={own.budget} display={figures.budget} />
            {!own.budget || !own.remainingBudget ? (
              noBudget
            ) : own.overAllocatedBy ? (
              <span className={styles.problem}>Over-allocated by {money(own.overAllocatedBy)}</span>
            ) : (
              <span>Remaining for campaigns {money(own.remainingBudget)}</span>
            )}
            {plan.allocation && <AllocationNote allocation={plan.allocation} />}
          </span>
        }
      />
      <SummaryCard
        label="Planned cost"
        value={moneyCompact(own.plannedCost)}
        fullValue={money(own.plannedCost)}
        tip="The planned cost of every activity in the plan, added up."
        detail={
          own.plannedCost ? (
            <Beside own={own.plannedCost} display={figures.plannedCost} />
          ) : (
            <span>Nothing planned yet</span>
          )
        }
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
        value={
          figures.activities === 0 ? '—' : `${figures.activitiesApproved} of ${figures.activities}`
        }
        tip="Activities that are Approved, In progress or Complete, out of all activities in the plan."
        detail={<span>{APPROVAL_PROGRESS[figures.approvalProgress].word}</span>}
      />
    </section>
  );
}

/** "≈ 759,493.67 USD" under a plan-currency amount when the reader's currency differs (016). */
function Beside({ own, display }: { own: Money | null; display: Money | null }) {
  if (!differentCurrency(own, display) || !display) {
    return null;
  }
  return <span className={styles.beside}>≈ {formatMoneyText(display)}</span>;
}

/**
 * "Allocated from 'FY2026 Global marketing budget' — Carlos Mendes" (014 FR-031): where the
 * budget comes from, the global budget a link for those who may open it.
 */
function AllocationNote({ allocation }: { allocation: PlanAllocation }) {
  const name = `'${allocation.globalBudget.name}'`;
  return (
    <span className={styles.allocation}>
      Allocated from{' '}
      {allocation.canOpen ? (
        <Link to={`/global-budgets/${encodeURIComponent(allocation.globalBudget.id)}`}>{name}</Link>
      ) : (
        name
      )}
      {allocation.owner && ` — ${allocation.owner.name}`}
    </span>
  );
}
