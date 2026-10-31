import { Link } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { ViewOnlyArea } from '@/ui/ViewOnlyNote/view-only-area';
import { AmountPair } from './AmountPair';
import { PinButton } from './PinButton';
import styles from './PlanCard.module.css';
import { BusinessUnitText, PlanCompleteness, PlanStatusChip } from './PlanFacts';
import type { PlanSummary } from './api';
import { changesNothingIn } from './plan-access';
import { activitiesApprovedText } from './plan-text';
import { PlanTypeTag } from './PlanTypeTag';
import { BcpirTag } from './BcpirTag';
import { FoundIn } from './FoundIn';

export interface PlanCardProps {
  plan: PlanSummary;
  actions: ActionsMenuItem[];
  pinned: boolean;
  onPinToggle: () => void;
}

export function PlanCard({ plan, actions, pinned, onPinToggle }: PlanCardProps) {
  const { figures } = plan;
  const used = figures.budgetUsedPercent;
  return (
    <article className={styles.card} aria-labelledby={`plan-${plan.id}`}>
      <div className={styles.top}>
        <div className={styles.title}>
          <h2 className={styles.name} id={`plan-${plan.id}`}>
            <Link to={`/plans/${encodeURIComponent(plan.id)}`}>{plan.name}</Link>
          </h2>
          <p className={styles.id}>
            <span>
              {plan.id} · {plan.year}
              {plan.aop && ' · AOP'}
            </span>
            <PlanTypeTag plan={plan} />
            <BcpirTag plan={plan} />
          </p>
          <FoundIn foundIn={plan.foundIn} />
        </div>
        <div className={styles.topActions}>
          <PinButton planName={plan.name} pinned={pinned} onToggle={onPinToggle} />
          <ViewOnlyArea active={changesNothingIn(plan.access)}>
            <ActionsMenu label={plan.name} items={actions} />
          </ViewOnlyArea>
        </div>
      </div>
      <div className={styles.chips}>
        <PlanStatusChip status={plan.status} />
        <PlanCompleteness planId={plan.id} completeness={plan.completeness} />
      </div>
      <div className={styles.row}>
        {plan.owner ? (
          <span className={styles.owner}>
            <span className="visually-hidden">Owner: </span>
            <PersonLabel person={plan.owner} />
          </span>
        ) : (
          <span className={styles.muted}>No owner yet</span>
        )}
      </div>
      <div className={styles.facts}>
        {plan.teams.length > 0 && <TeamPills teams={plan.teams} />}
        <BusinessUnitText units={plan.businessUnit} />
      </div>
      <div className={styles.row}>
        {figures.budget ? (
          <ProgressBar
            value={used}
            label="Budget used"
            tone={figures.overBudget ? 'problem' : undefined}
          />
        ) : (
          <span className={styles.muted}>No plan budget set</span>
        )}
      </div>
      <dl className={styles.money}>
        <div>
          <dt>Planned cost</dt>
          <dd>
            <AmountPair
              primary={plan.planPlannedCost ?? figures.plannedCost}
              secondary={figures.plannedCost}
              inline
            />
          </dd>
        </div>
        <div>
          <dt>Plan budget</dt>
          <dd>
            {figures.budget ? (
              <>
                <AmountPair
                  primary={plan.planBudget ?? figures.budget}
                  secondary={figures.budget}
                  inline
                />{' '}
                {used !== null && figures.plannedCost && (
                  <span className={figures.overBudget ? styles.problem : styles.used}>
                    {used}% used
                  </span>
                )}
              </>
            ) : (
              '—'
            )}
          </dd>
        </div>
      </dl>
      <p className={styles.approved}>
        {activitiesApprovedText(figures.activitiesApproved, figures.activities)}
      </p>
    </article>
  );
}
