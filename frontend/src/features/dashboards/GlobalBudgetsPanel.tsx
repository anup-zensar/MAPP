import { Link } from 'react-router';
import { useFormatters } from '@/format/useFormatters';
import { Card } from '@/ui/Card/Card';
import type { DashboardGlobalBudget } from './api';
import styles from './Dashboards.module.css';

interface GlobalBudgetsPanelProps {
  budgets: DashboardGlobalBudget[];
}

export function GlobalBudgetsPanel({ budgets }: GlobalBudgetsPanelProps) {
  const { money } = useFormatters();
  return (
    <Card title="Global budgets" subtitle="As the Global budgets page counts them">
      {budgets.length === 0 ? (
        <p className={styles.note}>No global budgets for this year.</p>
      ) : (
        <div className={styles.scroll}>
          <table className={styles.table}>
            <caption>Global budgets of the year and their allocations</caption>
            <thead>
              <tr>
                {['Global budget', 'Amount', 'Allocated', 'Unallocated', 'Planned cost'].map(
                  (title, i) => (
                    <th key={title} scope="col" className={i > 0 ? styles.number : undefined}>
                      {title}
                    </th>
                  ),
                )}
              </tr>
            </thead>
            <tbody>
              {budgets.map((b) => (
                <tr key={b.id}>
                  <th scope="row">
                    <Link to={`/budgets/${b.id}`}>{b.name}</Link>
                  </th>
                  <td className={styles.number}>{money(b.amount)}</td>
                  <td className={styles.number}>{money(b.allocated)}</td>
                  <td className={styles.number}>
                    {money(b.unallocated)}
                    {b.overAllocated && <span className={styles.problem}> · Over-allocated</span>}
                  </td>
                  <td className={styles.number}>{money(b.plannedCost)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
