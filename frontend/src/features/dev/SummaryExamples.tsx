import { ClipboardList, ListChecks } from 'lucide-react';
import { Card } from '@/ui/Card/Card';
import { LinkSummaryCard } from '@/ui/SummaryCard/LinkSummaryCard';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { useFormatters } from '@/format/useFormatters';
import styles from './ComponentGalleryPage.module.css';

export function SummaryExamples() {
  // Amounts as the signed-in person shows them (feature 003, FR-023).
  const { money, moneyCompact } = useFormatters();
  return (
    <>
      <Card title="SummaryCard">
        <div className={styles.summaries}>
          <SummaryCard label="Plans" value={6} tip="The plans of this year that you can see." />
          <SummaryCard
            label="Planned cost"
            value={moneyCompact({ amount: '1250000', currency: 'USD' })}
            fullValue={money({ amount: '1250000', currency: 'USD' })}
            tip="The planned cost of every activity in these plans."
          />
          <SummaryCard
            label="Waiting for approval"
            value={3}
            tone="review"
            tip="Activities sent for approval that are not approved or declined yet."
          />
          <SummaryCard
            label="Over budget"
            value={money({ amount: '-12500', currency: 'USD' })}
            tone="problem"
            tip="How much the planned cost is above the budget."
          />
        </div>
      </Card>

      <Card title="LinkSummaryCard" subtitle="The link variant: the whole card opens its list">
        <div className={styles.summaries}>
          <LinkSummaryCard
            icon={<ClipboardList />}
            tint="var(--info)"
            label="My teams' plans"
            tip="Plans that have at least one of your teams."
            value="3"
            detail={<span>2 published · 1 draft</span>}
            to="/plans?scope=myTeams"
            linkLabel="open the Plans list"
          />
          <LinkSummaryCard
            icon={<ListChecks />}
            tint="var(--level-campaign)"
            label="Planned cost"
            tip="The planned cost of every activity in these plans."
            value={moneyCompact({ amount: '1250000', currency: 'USD' })}
            fullValue={money({ amount: '1250000', currency: 'USD' })}
            to="#main"
            linkLabel="go to the top of this page"
          />
        </div>
      </Card>
    </>
  );
}
