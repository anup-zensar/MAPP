import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { useFormatters } from '@/format/useFormatters';
import type { FinancialTransactionPage } from './api';
import { accruedLabel, TILE_TIPS } from './financials-text';
import styles from './FinancialsPage.module.css';

/** Committed, Invoiced, Paid and Accrued for the filters, whatever the tab (FR-021). */
export function FinancialTotals({
  page,
  failed,
}: {
  page: FinancialTransactionPage | undefined;
  failed: boolean;
}) {
  const { money, moneyCompact, dateFormat } = useFormatters();
  if (failed) {
    return null;
  }
  if (!page) {
    return (
      <div className={styles.cards} aria-hidden="true">
        {[0, 1, 2, 3].map((index) => (
          <div key={index} className={styles.cardSkeleton}>
            <Skeleton lines={2} height="20px" />
          </div>
        ))}
      </div>
    );
  }
  const { totals, source } = page;
  const card = (label: string, value: (typeof totals)['committed'], tip: string) => (
    <SummaryCard
      label={label}
      value={moneyCompact(value)}
      fullValue={money(value)}
      tip={tip}
      tone={value.amount.startsWith('-') ? 'problem' : undefined}
    />
  );

  return (
    <div className={styles.cards}>
      {card('Committed', totals.committed, TILE_TIPS.committed)}
      {card('Invoiced', totals.invoiced, TILE_TIPS.invoiced)}
      {card('Paid', totals.paid, TILE_TIPS.paid)}
      {card(
        accruedLabel(source.accrualMonth, dateFormat),
        totals.accrued,
        TILE_TIPS.accrued(source.accrualMonth, dateFormat),
      )}
    </div>
  );
}
