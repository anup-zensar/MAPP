import { ClipboardList } from 'lucide-react';
import type { MouseEvent } from 'react';
import { Link, useNavigate } from 'react-router';
import { Button, LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { TeamPills } from '@/ui/TeamPill/TeamPill';
import { STATUS } from '@/features/plans';
import { cx } from '@/ui/cx';
import { formatWhole } from '@/format/decimal';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { useHomePlans } from './api';
import type { HomePlanRow, HomeScope } from './api';
import { plansListHref } from './home-links';
import {
  NOTHING_PLANNED,
  PLANS_AT_A_GLANCE,
  approvedPercent,
  approvedText,
  isZero,
  percentText,
  viewAllPlansText,
} from './home-text';
import styles from './PlansAtAGlanceCard.module.css';
import { SectionError } from './SectionError';

export interface PlansAtAGlanceCardProps {
  scope: HomeScope;
  /** Widens Home to every plan (the Switch to Everything of the empty state, FR-021). */
  onShowEverything: () => void;
}

/**
 * Plans at a glance (FR-008 – FR-010): each plan in scope with its status, teams, counts, approval
 * progress, budget used and planned cost, newest year first; at most 10, with "View all <n>
 * plans". The table follows the data table standard (design system §5.4; UI audit X-18): a grey
 * header band in sentence case, numbers right-aligned under their headings, and a shadow at the
 * edge with more columns when it scrolls sideways inside the card.
 */
export function PlansAtAGlanceCard({ scope, onShowEverything }: PlansAtAGlanceCardProps) {
  const plans = useHomePlans(scope);

  return (
    <Card
      title="Plans at a glance"
      subtitle={PLANS_AT_A_GLANCE.subtitle[scope]}
      actions={<InfoTip label="Plans at a glance" text={PLANS_AT_A_GLANCE.tip} />}
    >
      {plans.isPending ? (
        <Skeleton lines={4} height="18px" />
      ) : plans.isError ? (
        <SectionError onRetry={() => void plans.refetch()} />
      ) : plans.data.items.length === 0 ? (
        <EmptyState
          icon={<ClipboardList />}
          title={PLANS_AT_A_GLANCE.empty[scope]}
          action={
            scope === 'myTeams' ? (
              <Button onClick={onShowEverything}>Switch to Everything</Button>
            ) : (
              <LinkButton to="/plans">Go to Plans</LinkButton>
            )
          }
        />
      ) : (
        <>
          <PlansTable rows={plans.data.items} />
          {plans.data.total > plans.data.items.length && (
            <p className={styles.more}>
              <Link to={plansListHref(scope)}>{viewAllPlansText(plans.data.total)}</Link>
            </p>
          )}
        </>
      )}
    </Card>
  );
}

function PlansTable({ rows }: { rows: HomePlanRow[] }) {
  const navigate = useNavigate();
  const { money } = useFormatters();

  /** A click anywhere on a row opens the plan; its links keep working on their own. */
  function openRow(event: MouseEvent<HTMLTableRowElement>, href: string) {
    if (!(event.target instanceof Element && event.target.closest('a, button'))) {
      navigate(href);
    }
  }

  return (
    <div className={styles.scroll}>
      <table className={styles.table}>
        <caption className="visually-hidden">Plans at a glance</caption>
        <thead>
          <tr>
            <th scope="col">Plan</th>
            <th scope="col">Status</th>
            <th scope="col">Teams</th>
            <th scope="col" className={styles.number}>
              Campaigns
            </th>
            <th scope="col" className={styles.number}>
              Activities
            </th>
            <th scope="col">Approval</th>
            <th scope="col">Budget used</th>
            <th scope="col" className={styles.number}>
              Planned cost
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((plan) => {
            const href = recordHref('plan', plan.id);
            const status = STATUS[plan.status];
            return (
              <tr key={plan.id} className={styles.row} onClick={(event) => openRow(event, href)}>
                <th scope="row">
                  <Link to={href} className={styles.name}>
                    {plan.name}
                  </Link>
                  <span className={styles.meta}>
                    {plan.id} · {plan.year}
                    {plan.strategicAop && (
                      <>
                        {' '}
                        {/* A tag, not a status: blue chips mean "running" (design system §4.1). */}
                        <span className={styles.aop}>AOP</span>
                      </>
                    )}
                  </span>
                </th>
                <td>
                  <StatusChip tone={status.tone}>{status.word}</StatusChip>
                </td>
                <td>
                  {/* Each team in its own colour, as TeamRef shows it (D-201, D-217). */}
                  <TeamPills teams={plan.teams} />
                </td>
                <td className={styles.number}>{formatWhole(plan.campaignCount)}</td>
                <td className={styles.number}>{formatWhole(plan.activityCount)}</td>
                <td>
                  {plan.activityCount === 0 ? (
                    <span className={styles.muted}>No activities</span>
                  ) : (
                    <span className={styles.bar}>
                      <ProgressBar
                        value={approvedPercent(plan.approvedActivityCount, plan.activityCount)}
                        label="Approval progress"
                      />
                      <span>{approvedText(plan.approvedActivityCount, plan.activityCount)}</span>
                    </span>
                  )}
                </td>
                <td>
                  {plan.budgetUsedPercent === null ? (
                    <span className={styles.muted}>
                      —
                      {plan.budget === null && (
                        <>
                          {' · '}
                          <span className={styles.note}>No budget</span>
                        </>
                      )}
                    </span>
                  ) : (
                    <span className={styles.bar}>
                      <ProgressBar
                        value={plan.budgetUsedPercent}
                        label="Budget used"
                        tone={plan.overBudget ? 'problem' : undefined}
                      />
                      <span className={cx(plan.overBudget && styles.problem)}>
                        {percentText(plan.budgetUsedPercent)}
                      </span>
                    </span>
                  )}
                </td>
                <td className={styles.number}>
                  {plan.activityCount === 0 || isZero(plan.plannedCost) ? (
                    // Nothing planned is "—", never "$0.00" (design system §4.4).
                    <span className={styles.muted}>
                      —<span className="visually-hidden"> {NOTHING_PLANNED}</span>
                    </span>
                  ) : (
                    money(plan.plannedCost)
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
