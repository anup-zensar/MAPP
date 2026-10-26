import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { formatWhole } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { negativeBalanceText } from './all-work-text';
import type { AllWorkSummary } from './all-work-types';
import styles from './SummaryCards.module.css';

/** The cards' tips (plan "User-visible messages"). */
const TIPS = {
  campaigns: 'Campaigns that match the filters. Campaigns shown only for context are not counted.',
  initiatives:
    'Initiatives that match the filters. Initiatives shown only for context are not counted.',
  activities: 'Activities that match the filters and search.',
  plannedCost:
    "The sum of the matching activities' planned cost. An activity without a planned cost counts as zero.",
  planBudget: 'The sum of the budgets of the matching plans.',
  unassigned:
    "Planned cost not yet split into line items: the sum of the matching activities' positive balances.",
};

export interface SummaryCardsProps {
  /** null while the first answer loads. */
  summary: AllWorkSummary | null;
}

/**
 * The six summary cards of All work (FR-003), which follow every filter and search: counts, and
 * money in compact form with the full value on hover and focus. The Unassigned planned cost card
 * says in red how many activities have line items above their planned cost.
 */
export function SummaryCards({ summary }: SummaryCardsProps) {
  const { money, moneyCompact } = useFormatters();

  if (!summary) {
    return (
      <section className={styles.cards} aria-label="Summary">
        {[0, 1, 2, 3, 4, 5].map((card) => (
          <div key={card} className={styles.skeleton}>
            <Skeleton width="60%" />
            <Skeleton height="24px" width="40%" />
          </div>
        ))}
      </section>
    );
  }

  return (
    <section className={styles.cards} aria-label="Summary">
      <SummaryCard label="Campaigns" value={formatWhole(summary.campaigns)} tip={TIPS.campaigns} />
      <SummaryCard
        label="Initiatives"
        value={formatWhole(summary.initiatives)}
        tip={TIPS.initiatives}
      />
      <SummaryCard
        label="Activities"
        value={formatWhole(summary.activities)}
        tip={TIPS.activities}
      />
      <SummaryCard
        label="Planned cost"
        value={moneyCompact(summary.plannedCost)}
        fullValue={money(summary.plannedCost)}
        tip={TIPS.plannedCost}
      />
      <SummaryCard
        label="Plan budget"
        value={moneyCompact(summary.planBudget)}
        fullValue={money(summary.planBudget)}
        tip={TIPS.planBudget}
      />
      <SummaryCard
        label="Unassigned planned cost"
        value={moneyCompact(summary.unassignedPlannedCost)}
        fullValue={money(summary.unassignedPlannedCost)}
        tip={TIPS.unassigned}
        detail={
          summary.negativeBalanceCount > 0 && (
            <span className={styles.problem}>
              {negativeBalanceText(summary.negativeBalanceCount)}
            </span>
          )
        }
      />
    </section>
  );
}
