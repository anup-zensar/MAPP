import { Link } from 'react-router';
import { useFormatters } from '@/format/useFormatters';
import { Card } from '@/ui/Card/Card';
import type { DashboardCampaignRow } from './api';
import styles from './Dashboards.module.css';

interface CampaignListsProps {
  overBudget: DashboardCampaignRow[];
  leftToPlan: DashboardCampaignRow[];
}

export function CampaignLists({ overBudget, leftToPlan }: CampaignListsProps) {
  return (
    <>
      <CampaignList
        title="Campaigns most over budget"
        difference="Over budget by"
        rows={overBudget}
      />
      <CampaignList
        title="Campaigns with the most left to plan"
        difference="Left to plan"
        rows={leftToPlan}
      />
    </>
  );
}

interface CampaignListProps {
  title: string;
  difference: string;
  rows: DashboardCampaignRow[];
}

function CampaignList({ title, difference, rows }: CampaignListProps) {
  const { money } = useFormatters();
  return (
    <Card title={title} subtitle="Up to five campaigns with a budget">
      {rows.length === 0 ? (
        <p className={styles.note}>None.</p>
      ) : (
        <div className={styles.scroll}>
          <table className={styles.table}>
            <caption>{title}</caption>
            <thead>
              <tr>
                {['Campaign', 'Plan', 'Budget', 'Planned cost', difference].map((t, i) => (
                  <th key={t} scope="col" className={i > 1 ? styles.number : undefined}>
                    {t}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.campaign.id}>
                  <th scope="row">
                    <Link to={`/plans/${row.plan.id}/campaigns/${row.campaign.id}`}>
                      {row.campaign.name}
                    </Link>
                  </th>
                  <td>{row.plan.name}</td>
                  <td className={styles.number}>{money(row.figures.budget)}</td>
                  <td className={styles.number}>{money(row.figures.plannedCost)}</td>
                  <td className={styles.number}>{money(row.difference)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
