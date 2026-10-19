import { Link } from 'react-router';
import { ActionsMenu, StatusChip, TeamPills } from '@/components';
import type { ActionsMenuItem } from '@/components';
import styles from './PlanCard.module.css';
import {
  BudgetUsed,
  BusinessUnitText,
  PlanCompleteness,
  PlanOwner,
  PlanStatusChip,
} from './PlanFacts';
import type { PlanSummary } from './plansApi';
import { activitiesApprovedText } from './planText';

export interface PlanCardProps {
  plan: PlanSummary;
  actions: ActionsMenuItem[];
}

/**
 * One plan on the Plans list (FR-002): its name (opening the plan page, never the form), ID and
 * year, status, completeness and AOP chips, owner, teams, business unit, the budget used, the
 * activities approved and the ⋯ Actions menu.
 */
export function PlanCard({ plan, actions }: PlanCardProps) {
  const { figures } = plan;
  return (
    <article className={styles.card} aria-labelledby={`plan-${plan.id}`}>
      <div className={styles.top}>
        <div className={styles.title}>
          <h2 className={styles.name} id={`plan-${plan.id}`}>
            <Link to={`/plans/${encodeURIComponent(plan.id)}`}>{plan.name}</Link>
          </h2>
          <p className={styles.id}>
            {plan.id} · {plan.year}
          </p>
        </div>
        <ActionsMenu label={plan.name} items={actions} />
      </div>
      <div className={styles.chips}>
        <PlanStatusChip status={plan.status} />
        <PlanCompleteness planId={plan.id} completeness={plan.completeness} />
        {plan.aop && <StatusChip tone="active">AOP</StatusChip>}
      </div>
      <div className={styles.facts}>
        <PlanOwner owner={plan.owner} />
        {plan.teams.length > 0 && <TeamPills teams={plan.teams} />}
        <BusinessUnitText units={plan.businessUnit} />
      </div>
      <BudgetUsed figures={figures} />
      <p className={styles.approved}>
        {activitiesApprovedText(figures.activitiesApproved, figures.activities)}
      </p>
    </article>
  );
}
