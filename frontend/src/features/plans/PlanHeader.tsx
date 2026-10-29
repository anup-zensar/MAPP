import { TriangleAlert } from 'lucide-react';
import type { ReactNode } from 'react';
import { FieldHelpTip } from '@/features/settings';
import { LinkButton } from '@/ui/Button/Button';
import { LevelTag } from '@/ui/LevelTag/LevelTag';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { useFormatters } from '@/format/useFormatters';
import { allows } from '@/session/gate-action';
import { BusinessUnitText, PlanCompleteness, PlanStatusChip } from './PlanFacts';
import styles from './PlanHeader.module.css';
import type { Plan } from './api';
import { yearDates } from './plan-text';
import { PlanTypeTag } from './PlanTypeTag';
import { BcpirTag } from './BcpirTag';
import { spaHref } from './spa-link';

export interface PlanHeaderProps {
  plan: Plan;
  statusAction?: ReactNode;
  moreItems: ActionsMenuItem[];
  extraFacts?: ReactNode;
}

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
          <LinkButton to={spaHref(plan.id)}>SP&amp;A view</LinkButton>
          {statusAction}
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
          <FieldHelpTip helpKey="plan.status" label="Status" />
          <span id="completeness" className={styles.anchor}>
            <PlanCompleteness planId={plan.id} completeness={plan.completeness} />
            <FieldHelpTip helpKey="common.completeness" label="Completeness" />
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
          <PlanTypeTag plan={plan} />
          <BcpirTag plan={plan} />
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
