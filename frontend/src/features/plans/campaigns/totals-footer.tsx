import type { TreeTableFooter } from '@/ui/TreeTable/TreeTable';
import type { Formatters } from '@/format/useFormatters';
import { AmountPair } from '../AmountPair';
import styles from './campaign-rows.module.css';
import type { PlanCampaigns } from './api';
import { ALLOCATED, totalsText } from './texts';

/**
 * The totals row (FR-022): "Total · 2 campaigns · 4 initiatives · 7 activities", the sum of the
 * campaign budgets ("Allocated") and the plan's planned cost.
 */
export function totalsFooter(data: PlanCampaigns, formatters: Formatters): TreeTableFooter {
  // Amounts in the plan currency, the reader's display currency beneath when it differs (016
  // FR-013).
  const { totals, displayTotals } = data;
  return {
    // "Total" on the first line, the counts in one line beneath (UI audit PL-05: never wrapping).
    name: (
      <span className={styles.total}>
        <span>Total</span>
        <span className={styles.left}>
          {totalsText(totals.campaigns, totals.initiatives, totals.activities)}
        </span>
      </span>
    ),
    budget: (
      <span className={styles.money}>
        <AmountPair
          primary={totals.allocatedToCampaigns}
          secondary={displayTotals.allocatedToCampaigns}
        />
        <span className={styles.left}>{ALLOCATED}</span>
      </span>
    ),
    plannedCost: totals.plannedCost ? (
      <AmountPair primary={totals.plannedCost} secondary={displayTotals.plannedCost} />
    ) : (
      formatters.money(null)
    ),
  };
}
