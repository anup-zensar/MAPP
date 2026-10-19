import { TriangleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { ActionsMenu, Breadcrumb, LevelTag, LinkButton, StatusChip, TeamPills } from '@/components';
import type { ActionsMenuItem } from '@/components';
import { localDate } from '@/lib/format';
import { useFormatters } from '@/lib/useFormatters';
import { BusinessUnitText, PlanCompleteness, PlanStatusChip } from './PlanFacts';
import styles from './PlanPage.module.css';
import type { Plan } from './plansApi';

export interface PlanHeaderProps {
  plan: Plan;
  /** Publish on a Draft plan, Unpublish on a Published one (feature 004, US5). */
  statusAction?: ReactNode;
  /** The ⋯ More menu: View history, Delete. */
  moreItems: ActionsMenuItem[];
}

/**
 * The plan's details header (FR-011, design system §6.3): breadcrumb, PLAN tag, name, status,
 * completeness and AOP chips, the key facts, who created and last changed it, and the actions.
 */
export function PlanHeader({ plan, statusAction, moreItems }: PlanHeaderProps) {
  const { date } = useFormatters();
  const owner = plan.owner;

  return (
    <header className={styles.header}>
      <Breadcrumb items={[{ label: 'Plans', to: '/plans' }, { label: plan.name }]} />
      <div className={styles.titleRow}>
        <div className={styles.titleBlock}>
          <LevelTag level="plan" />
          <h1 className={styles.title}>{plan.name}</h1>
          <div className={styles.chips}>
            <PlanStatusChip status={plan.status} />
            <PlanCompleteness planId={plan.id} completeness={plan.completeness} />
            {plan.aop && <StatusChip tone="active">AOP</StatusChip>}
          </div>
        </div>
        <div className={styles.actions}>
          <LinkButton to={`/plans/${encodeURIComponent(plan.id)}/edit`} variant="primary">
            Edit
          </LinkButton>
          {statusAction}
          <ActionsMenu label={plan.name} items={moreItems} />
        </div>
      </div>
      <ul className={styles.meta} aria-label="Plan facts">
        <li className={styles.id}>{plan.id}</li>
        <li>
          {owner ? (
            <>
              Owner: {owner.name}
              {!owner.active && ' (inactive)'}
            </>
          ) : (
            'No owner yet'
          )}
          {plan.ownerProblem && (
            <span className={styles.ownerProblem}>
              <TriangleAlert aria-hidden="true" />
              {plan.ownerProblem}
            </span>
          )}
        </li>
        <li>Year {plan.year}</li>
        {plan.teams.length > 0 && (
          <li>
            <TeamPills teams={plan.teams} linked />
          </li>
        )}
        <li>
          <BusinessUnitText units={plan.businessUnit} />
        </li>
      </ul>
      <p className={styles.changed}>
        Created by {plan.createdBy.name} on {date(localDate(plan.createdAt))}, last changed by{' '}
        {plan.updatedBy.name} on {date(localDate(plan.updatedAt))}
      </p>
    </header>
  );
}
