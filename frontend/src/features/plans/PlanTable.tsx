import { Link } from 'react-router';
import { ActionsMenu, ProgressBar, StatusChip, TeamPills } from '@/components';
import type { ActionsMenuItem } from '@/components';
import { BusinessUnitText, PlanCompleteness, PlanOwner, PlanStatusChip } from './PlanFacts';
import type { PlanSummary } from './plansApi';
import styles from './PlanTable.module.css';
import { activitiesApprovedText } from './planText';

export interface PlanTableProps {
  plans: PlanSummary[];
  actionsFor: (plan: PlanSummary) => ActionsMenuItem[];
}

/**
 * The Plans list as a table (FR-002): the same facts as the cards, one row per plan. On narrow
 * screens the table scrolls sideways inside its card, never the page.
 */
export function PlanTable({ plans, actionsFor }: PlanTableProps) {
  return (
    <div className={styles.scroll}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th scope="col">Name</th>
            <th scope="col">Year</th>
            <th scope="col">Status</th>
            <th scope="col">Completeness</th>
            <th scope="col">Owner</th>
            <th scope="col">Teams</th>
            <th scope="col">Business unit</th>
            <th scope="col">Budget used</th>
            <th scope="col">Activities approved</th>
            <th scope="col">
              <span className="visually-hidden">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {plans.map((plan) => {
            const { figures } = plan;
            return (
              <tr key={plan.id}>
                <th scope="row">
                  <Link to={`/plans/${encodeURIComponent(plan.id)}`} className={styles.name}>
                    {plan.name}
                  </Link>
                  <span className={styles.id}>{plan.id}</span>
                </th>
                <td className={styles.number}>{plan.year}</td>
                <td>
                  <span className={styles.chips}>
                    <PlanStatusChip status={plan.status} />
                    {plan.aop && <StatusChip tone="active">AOP</StatusChip>}
                  </span>
                </td>
                <td>
                  <PlanCompleteness planId={plan.id} completeness={plan.completeness} />
                </td>
                <td>
                  <PlanOwner owner={plan.owner} />
                </td>
                <td>{plan.teams.length > 0 ? <TeamPills teams={plan.teams} /> : '—'}</td>
                <td>
                  <BusinessUnitText units={plan.businessUnit} />
                </td>
                <td>
                  {figures.budget ? (
                    <span className={styles.used}>
                      <ProgressBar
                        value={figures.budgetUsedPercent}
                        label="Budget used"
                        tone={figures.overBudget ? 'problem' : undefined}
                      />
                      <span className={figures.overBudget ? styles.problem : undefined}>
                        {figures.budgetUsedPercent === null
                          ? '—'
                          : `${figures.budgetUsedPercent} %`}
                      </span>
                    </span>
                  ) : (
                    <span className={styles.muted}>No budget set</span>
                  )}
                </td>
                <td>{activitiesApprovedText(figures.activitiesApproved, figures.activities)}</td>
                <td>
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
