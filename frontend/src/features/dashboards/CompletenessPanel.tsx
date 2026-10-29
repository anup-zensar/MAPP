import { Card } from '@/ui/Card/Card';
import type { DashboardCompleteness } from './api';
import { percentText, word } from './dashboard-text';
import styles from './Dashboards.module.css';

interface CompletenessPanelProps {
  levels: DashboardCompleteness[];
}

export function CompletenessPanel({ levels }: CompletenessPanelProps) {
  return (
    <Card title="Plan details" subtitle="Records with every needed field filled in">
      <table className={styles.table}>
        <caption>Completeness by level</caption>
        <thead>
          <tr>
            {['Level', 'Records', 'All filled in', 'Share'].map((title, i) => (
              <th key={title} scope="col" className={i > 0 ? styles.number : undefined}>
                {title}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {levels.map((level) => (
            <tr key={level.level}>
              <th scope="row">{word(level.level)}</th>
              <td className={styles.number}>{level.total}</td>
              <td className={styles.number}>{level.complete}</td>
              <td className={styles.number}>{percentText(level.percent)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  );
}
