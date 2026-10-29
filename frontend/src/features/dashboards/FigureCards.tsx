import { useFormatters } from '@/format/useFormatters';
import type { DashboardFigures } from './api';
import { FINANCE_HIDDEN, percentText } from './dashboard-text';
import styles from './Dashboards.module.css';

interface FigureCardsProps {
  figures: DashboardFigures;
}

export function FigureCards({ figures }: FigureCardsProps) {
  const { money } = useFormatters();
  const overspent = figures.leftToPlan !== null && figures.leftToPlan.amount.startsWith('-');
  const cards = [
    { label: 'Budget', value: money(figures.budget), note: figures.budget ? '' : 'No budget set' },
    {
      label: 'Planned cost',
      value: money(figures.plannedCost),
      note: figures.overBudget
        ? 'Over budget'
        : `Budget used ${percentText(figures.budgetUsedPercent)}`,
      problem: figures.overBudget,
    },
    {
      label: 'Left to plan',
      value: money(figures.leftToPlan),
      note: overspent ? 'Planned cost is over the budget' : '',
      problem: overspent,
    },
    {
      label: 'Committed',
      value: money(figures.committed),
      note: figures.committed ? '' : FINANCE_HIDDEN,
    },
    {
      label: 'Actual (paid)',
      value: money(figures.actual),
      note: figures.actual
        ? `Budget spent ${percentText(figures.budgetSpentPercent)}`
        : FINANCE_HIDDEN,
    },
    { label: 'Unspent budget', value: money(figures.unspentBudget), note: '' },
  ];
  return (
    <dl className={styles.figures} aria-label="Summary">
      {cards.map((card) => (
        <div key={card.label} className={styles.figure}>
          <dt>{card.label}</dt>
          <dd className={styles.value}>{card.value}</dd>
          {card.note && (
            <dd className={card.problem ? styles.problem : styles.figureNote}>{card.note}</dd>
          )}
        </div>
      ))}
    </dl>
  );
}
