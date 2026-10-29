import { useFieldHelp } from '@/features/settings';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { formatWhole } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { negativeBalanceText } from './all-work-text';
import type { AllWorkSummary } from './all-work-types';
import styles from './SummaryCards.module.css';

export interface SummaryCardsProps {
  summary: AllWorkSummary | null;
}

/**
 * The six summary cards of All work, which follow every filter and search: counts, and money in
 * compact form with the full value on hover and focus.
 */
export function SummaryCards({ summary }: SummaryCardsProps) {
  const { money, moneyCompact } = useFormatters();
  const { help } = useFieldHelp();

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
      <SummaryCard
        label="Campaigns"
        value={formatWhole(summary.campaigns)}
        tip={help('allWork.campaigns')}
      />
      <SummaryCard
        label="Initiatives"
        value={formatWhole(summary.initiatives)}
        tip={help('allWork.initiatives')}
      />
      <SummaryCard
        label="Activities"
        value={formatWhole(summary.activities)}
        tip={help('allWork.activities')}
      />
      <SummaryCard
        label="Planned cost"
        value={moneyCompact(summary.plannedCost)}
        fullValue={money(summary.plannedCost)}
        tip={help('allWork.plannedCost')}
      />
      <SummaryCard
        label="Plan budget"
        value={moneyCompact(summary.planBudget)}
        fullValue={money(summary.planBudget)}
        tip={help('allWork.planBudget')}
      />
      <SummaryCard
        label="Unassigned planned cost"
        value={moneyCompact(summary.unassignedPlannedCost)}
        fullValue={money(summary.unassignedPlannedCost)}
        tip={help('allWork.unassignedPlannedCost')}
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
