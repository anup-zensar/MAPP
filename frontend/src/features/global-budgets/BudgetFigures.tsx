import { TriangleAlert } from 'lucide-react';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { cx } from '@/ui/cx';
import { useFormatters } from '@/format/useFormatters';
import type { GlobalBudgetFigures } from './globalBudgetsApi';
import { FIGURE_TIPS, overAllocatedText } from './globalBudgetsText';
import styles from './BudgetFigures.module.css';

/** The four figures with "Over-allocated by X" when Unallocated is below zero (D-015). */
function useFigureItems(figures: GlobalBudgetFigures) {
  const { money, moneyCompact } = useFormatters();
  const over = figures.overAllocated || figures.unallocated.amount.startsWith('-');
  const overBy = over
    ? money({ ...figures.unallocated, amount: figures.unallocated.amount.replace(/^-/, '') })
    : null;

  const item = (label: string, value: GlobalBudgetFigures['amount'], tip: string) => ({
    label,
    tip,
    value: money(value),
    short: moneyCompact(value),
    problem: false,
    detail: undefined as string | undefined,
  });

  return [
    item('Global budget', figures.amount, FIGURE_TIPS.amount),
    item('Allocated', figures.allocated, FIGURE_TIPS.allocated),
    {
      ...item('Unallocated', figures.unallocated, FIGURE_TIPS.unallocated),
      problem: over,
      detail: overBy ? overAllocatedText(overBy) : undefined,
    },
    item('Planned cost', figures.plannedCost, FIGURE_TIPS.plannedCost),
  ];
}

/**
 * Global budget, Allocated, Unallocated and Planned cost as the four summary cards of a global
 * budget's page (FR-032). Unallocated below zero is red with "Over-allocated by X" in words.
 */
export function BudgetSummaryCards({ figures }: { figures: GlobalBudgetFigures }) {
  const items = useFigureItems(figures);
  return (
    <div className={styles.cards}>
      {items.map((item) => (
        <SummaryCard
          key={item.label}
          label={item.label}
          value={item.short}
          fullValue={item.value}
          tip={item.tip}
          tone={item.problem ? 'problem' : undefined}
          detail={item.detail && <OverAllocated text={item.detail} />}
        />
      ))}
    </div>
  );
}

/** The same four figures inside a global budget's card on the list (FR-036). */
export function BudgetFigures({ figures }: { figures: GlobalBudgetFigures }) {
  const items = useFigureItems(figures);
  return (
    <dl className={styles.figures}>
      {items.map((item) => (
        <div key={item.label} className={styles.figure}>
          <dt className={styles.label}>{item.label}</dt>
          <dd className={cx(styles.value, item.problem && styles.problem)}>{item.value}</dd>
          {item.detail && (
            <dd>
              <OverAllocated text={item.detail} />
            </dd>
          )}
        </div>
      ))}
    </dl>
  );
}

function OverAllocated({ text }: { text: string }) {
  return (
    <span className={styles.over}>
      <TriangleAlert aria-hidden="true" />
      {text}
    </span>
  );
}
