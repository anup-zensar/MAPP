import { useFormatters } from '@/format/useFormatters';
import { Card } from '@/ui/Card/Card';
import type { DashboardApprovals } from './api';
import { NOTHING, word } from './dashboard-text';
import styles from './Dashboards.module.css';

interface ApprovalsPanelProps {
  approvals: DashboardApprovals;
}

export function ApprovalsPanel({ approvals }: ApprovalsPanelProps) {
  const { dateTime } = useFormatters();
  const none =
    approvals.waiting + approvals.approved + approvals.declined + approvals.autoApproved === 0;
  const stats = [
    { label: 'Waiting for a decision', value: String(approvals.waiting) },
    {
      label: 'Waiting since',
      value: approvals.oldestWaitingSince ? dateTime(approvals.oldestWaitingSince) : NOTHING,
    },
    { label: 'Approved', value: String(approvals.approved) },
    { label: 'Declined', value: String(approvals.declined) },
    { label: 'Approved automatically', value: String(approvals.autoApproved) },
    {
      label: 'Average time to decide',
      value: approvals.averageDecisionDays ? `${approvals.averageDecisionDays} days` : NOTHING,
    },
  ];
  return (
    <Card title="Approvals" subtitle="Activities and purchase requests of these plans">
      {none && <p className={styles.note}>No approvals yet.</p>}
      <dl className={styles.stats}>
        {stats.map((stat) => (
          <div key={stat.label}>
            <dt>{stat.label}</dt>
            <dd>{stat.value}</dd>
          </div>
        ))}
      </dl>
      <table className={styles.table}>
        <caption>Activities by approval status</caption>
        <thead>
          <tr>
            <th scope="col">Status</th>
            <th scope="col" className={styles.number}>
              Activities
            </th>
          </tr>
        </thead>
        <tbody>
          {approvals.activitiesByStatus.map((row) => (
            <tr key={row.status}>
              <th scope="row">{word(row.status)}</th>
              <td className={styles.number}>{row.count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  );
}
