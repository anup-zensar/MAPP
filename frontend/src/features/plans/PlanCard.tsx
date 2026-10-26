import { Link } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { AmountPair } from './AmountPair';
import { PinButton } from './PinButton';
import styles from './PlanCard.module.css';
import { BusinessUnitText, PlanCompleteness, PlanStatusChip } from './PlanFacts';
import type { PlanSummary } from './plansApi';
import { activitiesApprovedText } from './planText';
import { PlanTypeTag } from './PlanTypeTag';

export interface PlanCardProps {
  plan: PlanSummary;
  actions: ActionsMenuItem[];
  /** Whether this person pinned the plan (018 FR-003): the star is filled and stays visible. */
  pinned: boolean;
  /** Pins or unpins the plan for this person (018 FR-001). */
  onPinToggle: () => void;
}

/**
 * One plan on the Plans list (FR-002): its name (opening the plan page, never the form), ID, year,
 * type (016) and AOP; the status and one completeness chip; owner, teams and business unit; the
 * planned cost against the budget in the plan currency (016 FR-014); the activities approved; the
 * pin star (018) and the ⋯ Actions menu. Its seven rows share the grid rows of the cards beside
 * it, so owner, budget bar and footer line up across a row of cards (UI audit P-03).
 */
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
          </p>
        </div>
        <div className={styles.topActions}>
          <PinButton planName={plan.name} pinned={pinned} onToggle={onPinToggle} />
          <ActionsMenu label={plan.name} items={actions} />
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
