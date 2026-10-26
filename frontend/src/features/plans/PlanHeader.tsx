import { TriangleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { LinkButton } from '@/ui/Button/Button';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { useFormatters } from '@/lib/useFormatters';
import { allows } from './gateAction';
import { BusinessUnitText, PlanCompleteness, PlanStatusChip } from './PlanFacts';
import styles from './PlanHeader.module.css';
import type { Plan } from './plansApi';
import { yearDates } from './planText';
import { PlanTypeTag } from './PlanTypeTag';

export interface PlanHeaderProps {
  plan: Plan;
  /** Publish on a Draft plan, Unpublish on a Published one (feature 004, US5): a secondary button. */
  statusAction?: ReactNode;
  /** The ⋯ More menu: Copy, View history, Delete. */
  moreItems: ActionsMenuItem[];
  /** Facts added by later features, after the plan type and currency. */
  extraFacts?: ReactNode;
}

/**
 * The plan's header (FR-011, design system §2.3, §6.3) on the one page header of the app: the
 * breadcrumb, the plan's name, ⋯ More · Publish or Unpublish · **Edit** (the one primary, last),
 * and the meta row — level tag, status, one completeness chip, ID, owner, dates, teams and
 * business unit. Who created and last changed the plan is at the end of the Details tab.
 */
export function PlanHeader({ plan, statusAction, moreItems, extraFacts }: PlanHeaderProps) {
  const { dateRange } = useFormatters();
  const owner = plan.owner;

  return (
    <PageHeader
      breadcrumb={[{ label: 'Plans', to: '/plans' }, { label: plan.name }]}
      title={plan.name}
      moreActions={moreItems}
      actions={
        <>
          {statusAction}
          {/* 014 FR-011: Edit only for a person who may edit the plan. */}
          {allows(plan.access?.edit) && (
            <LinkButton to={`/plans/${encodeURIComponent(plan.id)}/edit`} variant="primary">
              Edit
            </LinkButton>
          )}
        </>
      }
      meta={
        <div className={styles.meta} aria-label="Plan facts" role="group">
          <LevelTag level="plan" />
          <PlanStatusChip status={plan.status} />
          {/* #completeness: where Home's "Show fields" leads (feature 009). */}
          <span id="completeness" className={styles.anchor}>
            <PlanCompleteness planId={plan.id} completeness={plan.completeness} />
          </span>
          <span className={styles.fact}>{plan.id}</span>
          <span className={styles.fact}>
            {owner ? <PersonLabel person={owner} /> : 'No owner yet'}
            {plan.ownerProblem && (
              <span className={styles.ownerProblem}>
                <TriangleAlert aria-hidden="true" />
                {plan.ownerProblem}
              </span>
            )}
          </span>
          <span className={styles.fact}>{yearDates(plan.year, dateRange)}</span>
          {/* 016 FR-032, FR-013: the plan type, and the plan currency when it is not USD. */}
          <PlanTypeTag plan={plan} />
          {plan.currency && plan.currency !== 'USD' && (
            <span className={styles.fact}>Planned in {plan.currency}</span>
          )}
          {plan.aop && <span className={styles.fact}>AOP plan</span>}
          {extraFacts}
          {plan.teams.length > 0 && <TeamPills teams={plan.teams} linked />}
          <span className={styles.fact}>
            <BusinessUnitText units={plan.businessUnit} />
          </span>
        </div>
      }
    />
  );
}
