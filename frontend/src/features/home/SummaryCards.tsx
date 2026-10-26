import { useCurrentUser } from '@/session/useSession';
import { ProgressBar } from '@/ui/ProgressBar/ProgressBar';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { cx } from '@/ui/cx';
import { formatWhole } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { useHomeSummary } from './homeApi';
import type { HomeScope, HomeSummary } from './homeApi';
import {
  NOTHING_PLANNED,
  PLANS_CARD,
  activityCountsText,
  isZero,
  ofPlanBudgetText,
  oldestWaitingText,
  percentUsedText,
  planCountsText,
  plansWithoutBudgetText,
} from './homeText';
import { myActivitiesHref, plansListHref, waitingForMeHref } from './homeLinks';
import { SectionError } from './SectionError';
import styles from './SummaryCards.module.css';
import { SummaryLinkCard } from './SummaryLinkCard';

/**
 * The four summary cards of Home (FR-005 – FR-007, design system §2.4 E): my teams' plans,
 * activities I own, waiting for my approval and planned cost against plan budget, each opening its
 * list with the same filter. Four in a row on wide screens, 2 × 2 below (UI audit H-06). While
 * loading each shows a placeholder; a failure shows "This section couldn't load".
 */
export function SummaryCards({ scope }: { scope: HomeScope }) {
  const person = useCurrentUser();
  const { money, moneyCompact } = useFormatters();
  const summary = useHomeSummary(scope);

  if (summary.isPending) {
    return (
      <section aria-label="Summary" className={styles.cards}>
        {[1, 2, 3, 4].map((card) => (
          <div key={card} className={styles.placeholder}>
            <Skeleton lines={3} />
          </div>
        ))}
      </section>
    );
  }
  if (summary.isError) {
    return (
      <section aria-label="Summary" className={styles.failed}>
        <SectionError onRetry={() => void summary.refetch()} />
      </section>
    );
  }

  const { plans, activitiesOwned, waitingForMe, plannedCost } = summary.data;
  const plansCard = PLANS_CARD[scope];
  // Nothing planned is "—", never "$0.00" (design system §4.4, functional review X-08).
  const nothingPlanned = plannedCost.planCount === 0 || isZero(plannedCost.plannedCost);

  return (
    <section aria-label="Summary" className={styles.cards}>
      <SummaryLinkCard
        label={plansCard.label}
        tip={plansCard.tip}
        value={formatWhole(plans.count)}
        detail={<span>{planCountsText(plans.published, plans.draft)}</span>}
        to={plansListHref(scope)}
        linkLabel="open the Plans list"
      />
      <SummaryLinkCard
        label="Activities I own"
        tip="Activities you own, in every plan."
        value={formatWhole(activitiesOwned.count)}
        detail={
          <span>{activityCountsText(activitiesOwned.declined, activitiesOwned.submitted)}</span>
        }
        to={myActivitiesHref(person.id)}
        linkLabel="open your activities"
      />
      <SummaryLinkCard
        label="Waiting for my approval"
        tip="Submitted activities waiting at a level where you are an approver."
        value={formatWhole(waitingForMe.count)}
        tone={waitingForMe.count > 0 ? 'review' : undefined}
        detail={<span>{oldestWaitingText(waitingForMe.oldestWaitingSince)}</span>}
        to={waitingForMeHref()}
        linkLabel="open Approvals"
      />
      <SummaryLinkCard
        label="Planned cost vs plan budget"
        tip="Planned cost of the plans in scope compared with their plan budgets."
        value={nothingPlanned ? '—' : moneyCompact(plannedCost.plannedCost)}
        fullValue={nothingPlanned ? undefined : money(plannedCost.plannedCost)}
        detail={
          plannedCost.planCount === 0 ? (
            <span>No plans in scope</span>
          ) : (
            <BudgetUsed
              cost={plannedCost}
              budget={moneyCompact(plannedCost.planBudget)}
              nothingPlanned={nothingPlanned}
            />
          )
        }
        to={plansListHref(scope)}
        linkLabel="open the Plans list"
      />
    </section>
  );
}

interface BudgetUsedProps {
  cost: HomeSummary['plannedCost'];
  budget: string;
  nothingPlanned: boolean;
}

/**
 * Card 4's lines (FR-006): "of <budget> plan budget", the budget-used bar with "<n>% used" (red
 * above the budget) and "Includes <n> plans with no budget"; without any plan budget "No plan
 * budgets set" and "—"; with nothing planned yet "Nothing planned yet" instead of the bar.
 */
function BudgetUsed({ cost, budget, nothingPlanned }: BudgetUsedProps) {
  const used = cost.budgetUsedPercent;
  return (
    <>
      {nothingPlanned && <span>{NOTHING_PLANNED}</span>}
      <span>{cost.planBudget ? ofPlanBudgetText(budget) : 'No plan budgets set'}</span>
      {!nothingPlanned && (
        <span className={styles.usage}>
          <ProgressBar
            value={used}
            label="Budget used"
            tone={cost.overBudget ? 'problem' : undefined}
          />
          <span className={cx(styles.percent, cost.overBudget && styles.problem)}>
            {used === null ? '—' : percentUsedText(used)}
          </span>
        </span>
      )}
      {cost.planBudget && cost.plansWithoutBudget > 0 && (
        <span>{plansWithoutBudgetText(cost.plansWithoutBudget)}</span>
      )}
    </>
  );
}
