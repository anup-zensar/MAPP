import { Link } from 'react-router';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { TeamPill } from '@/ui/TeamPill/TeamPill';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import type { TeamRef } from '@/ui/TeamPill/TeamPill';
import { AmountPair } from './AmountPair';
import { PinButton } from './PinButton';
import { PlanCompleteness, PlanStatusChip } from './PlanFacts';
import type { PlanSummary } from './plansApi';
import styles from './PlanTable.module.css';
import { PlanTypeTag } from './PlanTypeTag';

export interface PlanTableProps {
  plans: PlanSummary[];
  actionsFor: (plan: PlanSummary) => ActionsMenuItem[];
  /** Whether this person pinned the plan (018 FR-003): its star is filled and stays visible. */
  isPinned: (plan: PlanSummary) => boolean;
  /** Pins or unpins one plan for this person (018 FR-001). */
  onPinToggle: (plan: PlanSummary) => void;
}

/** Teams shown before "+n" in a table cell (design system §5.4: more than 2 show "+n"). */
const TEAMS_SHOWN = 2;

/**
 * The Plans list as a table (FR-002) on the data table standard (design system §5.4): the pin star
 * first (018); the name with its ID, year and type; at most two status chips; the owner and teams
 * on one line; money right-aligned under its header in the plan currency, the display currency
 * beside when it differs (016 FR-014); no cell repeats its column's label (UI audit P-02). On
 * narrow screens the table scrolls sideways inside its card, never the page.
 */
export function PlanTable({ plans, actionsFor, isPinned, onPinToggle }: PlanTableProps) {
  return (
    <div className={styles.scroll}>
      <table className={styles.table} aria-label="Plans">
        <thead>
          <tr>
            <th scope="col" className={styles.pin}>
              <span className="visually-hidden">Pinned</span>
            </th>
            <th scope="col" className={styles.nameColumn}>
              Name
            </th>
            <th scope="col">Status</th>
            <th scope="col" className={styles.personColumn}>
              Owner
            </th>
            <th scope="col">Teams</th>
            <th scope="col" className={styles.number}>
              Plan budget
            </th>
            <th scope="col" className={styles.number}>
              Planned cost
            </th>
            <th scope="col" className={styles.number}>
              Approved
            </th>
            <th scope="col" className={styles.actions}>
              <span className="visually-hidden">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {plans.map((plan) => {
            const { figures } = plan;
            const used = figures.budgetUsedPercent;
            return (
              <tr key={plan.id}>
                <td className={styles.pin}>
                  <PinButton
                    planName={plan.name}
                    pinned={isPinned(plan)}
                    onToggle={() => onPinToggle(plan)}
                  />
                </td>
                <th scope="row" className={styles.nameColumn}>
                  <Link to={`/plans/${encodeURIComponent(plan.id)}`} className={styles.name}>
                    {plan.name}
                  </Link>
                  <span className={styles.sub}>
                    <span>
                      {plan.id} · {plan.year}
                      {plan.aop && ' · AOP'}
                    </span>
                    <PlanTypeTag plan={plan} />
                  </span>
                </th>
                <td>
                  <span className={styles.chips}>
                    <PlanStatusChip status={plan.status} />
                    <PlanCompleteness planId={plan.id} completeness={plan.completeness} compact />
                  </span>
                </td>
                <td className={styles.personColumn}>
                  {plan.owner ? (
                    <PersonLabel person={plan.owner} />
                  ) : (
                    <span className={styles.muted}>No owner yet</span>
                  )}
                </td>
                <td>
                  <Teams teams={plan.teams} />
                </td>
                <td className={styles.number}>
                  {figures.budget ? (
                    <AmountPair
                      primary={plan.planBudget ?? figures.budget}
                      secondary={figures.budget}
                    />
                  ) : (
                    <span className={styles.muted}>—</span>
                  )}
                </td>
                <td className={styles.number}>
                  <span className={styles.stack}>
                    <AmountPair
                      primary={plan.planPlannedCost ?? figures.plannedCost}
                      secondary={figures.plannedCost}
                    />
                    {used !== null && figures.plannedCost && (
                      <span className={figures.overBudget ? styles.problem : styles.detail}>
                        {used}% used
                      </span>
                    )}
                  </span>
                </td>
                <td className={styles.number}>
                  {figures.activities === 0 ? (
                    <span className={styles.muted}>—</span>
                  ) : (
                    <span>
                      {figures.activitiesApproved} of {figures.activities}
                    </span>
                  )}
                </td>
                <td className={styles.actions}>
                  <ActionsMenu label={plan.name} items={actionsFor(plan)} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

/** At most two team tags on one line, then "+n" with the other names in a tooltip. */
function Teams({ teams }: { teams: TeamRef[] }) {
  if (teams.length === 0) {
    return <span className={styles.muted}>—</span>;
  }
  const shown = teams.slice(0, TEAMS_SHOWN);
  const rest = teams.slice(TEAMS_SHOWN);
  return (
    <span className={styles.teams}>
      {shown.map((team) => (
        <TeamPill key={team.id} team={team} />
      ))}
      {rest.length > 0 && (
        <Tooltip content={rest.map((team) => team.name).join(', ')}>
          <button
            type="button"
            className={styles.more}
            aria-label={`+${rest.length} more: ${rest.map((team) => team.name).join(', ')}`}
          >
            +{rest.length}
          </button>
        </Tooltip>
      )}
    </span>
  );
}
