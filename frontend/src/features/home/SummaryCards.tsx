import { ClipboardList, Hourglass, ListChecks, Wallet } from 'lucide-react';
import { useCurrentUser } from '@/app/useSession';
import { LinkSummaryCard, ProgressBar, Skeleton } from '@/components';
import { cx } from '@/lib/cx';
import { formatWhole } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import { useHomeSummary } from './homeApi';
import type { HomeScope, HomeSummary } from './homeApi';
import {
  PLANS_CARD,
  activityCountsText,
  ofPlanBudgetText,
  oldestWaitingText,
  percentUsedText,
  planCountsText,
  plansWithoutBudgetText,
} from './homeText';
import { myActivitiesHref, plansListHref, waitingForMeHref } from './homeLinks';
import { SectionError } from './SectionError';
import styles from './SummaryCards.module.css';

/**
 * The four summary cards of Home (FR-005 – FR-007): my teams' plans, activities I own, waiting for
 * my approval and planned cost against plan budget, each opening its list with the same filter.
 * While loading each shows a placeholder; a failure shows "This section couldn't load".
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
  const hasPlans = plannedCost.planCount > 0;

  return (
    <section aria-label="Summary" className={styles.cards}>
      <LinkSummaryCard
        icon={<ClipboardList />}
        tint="var(--blue)"
        label={plansCard.label}
        tip={plansCard.tip}
        value={formatWhole(plans.count)}
        detail={<span>{planCountsText(plans.published, plans.draft)}</span>}
        to={plansListHref(scope)}
        linkLabel="open the Plans list"
      />
      <LinkSummaryCard
        icon={<ListChecks />}
        tint="var(--level-campaign)"
        label="Activities I own"
        tip="Activities you own, in every plan."
        value={formatWhole(activitiesOwned.count)}
        detail={
          <span>{activityCountsText(activitiesOwned.declined, activitiesOwned.submitted)}</span>
        }
        to={myActivitiesHref(person.id)}
        linkLabel="open your activities"
      />
      <LinkSummaryCard
        icon={<Hourglass />}
        tint="var(--status-review)"
        label="Waiting for my approval"
        tip="Submitted activities waiting at a level where you are an approver."
        value={formatWhole(waitingForMe.count)}
        tone={waitingForMe.count > 0 ? 'review' : undefined}
        detail={<span>{oldestWaitingText(waitingForMe.oldestWaitingSince)}</span>}
        to={waitingForMeHref()}
        linkLabel="open Approvals"
      />
      <LinkSummaryCard
        icon={<Wallet />}
        tint="var(--status-positive)"
        label="Planned cost vs plan budget"
        tip="Planned cost of the plans in scope compared with their plan budgets."
        value={hasPlans ? moneyCompact(plannedCost.plannedCost) : '—'}
        fullValue={hasPlans ? money(plannedCost.plannedCost) : undefined}
        detail={
          hasPlans ? (
            <BudgetUsed cost={plannedCost} budget={moneyCompact(plannedCost.planBudget)} />
          ) : (
            <span>No plans in scope</span>
          )
        }
        to={plansListHref(scope)}
        linkLabel="open the Plans list"
      />
    </section>
  );
}

/**
 * Card 4's lines (FR-006): "of <budget> plan budget", the budget-used bar with "<n> % used" (red
 * above the budget) and "Includes <n> plans with no budget"; without any plan budget "No plan
 * budgets set" and "—".
 */
function BudgetUsed({ cost, budget }: { cost: HomeSummary['plannedCost']; budget: string }) {
  const used = cost.budgetUsedPercent;
  return (
    <>
      <span>{cost.planBudget ? ofPlanBudgetText(budget) : 'No plan budgets set'}</span>
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
      {cost.planBudget && cost.plansWithoutBudget > 0 && (
        <span>{plansWithoutBudgetText(cost.plansWithoutBudget)}</span>
      )}
    </>
  );
}
