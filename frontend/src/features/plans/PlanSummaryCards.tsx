import { Link } from 'react-router';
import type { components } from '@/api/generated/schema';
import { useFieldHelp } from '@/features/settings';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { formatMoneyText } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { differentCurrency } from './plan-currency';
import type { Plan } from './api';
import styles from './PlanSummaryCards.module.css';
import { APPROVAL_PROGRESS } from './plan-text';

type Money = components['schemas']['Money'];
type PlanAllocation = components['schemas']['PlanAllocation'];

export interface PlanSummaryCardsProps {
  plan: Plan;
}

/** The plan's four summary cards, in the plan currency with the display currency beneath. */
export function PlanSummaryCards({ plan }: PlanSummaryCardsProps) {
  const { money, moneyCompact } = useFormatters();
  const { help } = useFieldHelp();
  const { figures } = plan;
  const own = plan.planFigures ?? figures;
  const noBudget = <span>No plan budget set</span>;

  return (
    // #budget: where Home's "Open budget" leads (feature 009).
    <section id="budget" className={styles.summary} aria-label="Summary">
      <SummaryCard
        label="Plan budget"
        value={moneyCompact(own.budget)}
        fullValue={money(own.budget)}
        tip={help('plan.budget')}
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
        tip={help('plan.plannedCost')}
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
        tip={help('plan.budgetUsed')}
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
        tip={help('plan.activitiesApproved')}
        detail={<span>{APPROVAL_PROGRESS[figures.approvalProgress].word}</span>}
      />
    </section>
  );
}

function Beside({ own, display }: { own: Money | null; display: Money | null }) {
  if (!differentCurrency(own, display) || !display) {
    return null;
  }
  return <span className={styles.beside}>≈ {formatMoneyText(display)}</span>;
}

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
