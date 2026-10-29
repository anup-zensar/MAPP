import { Link } from 'react-router';
import { Card } from '@/ui/Card/Card';
import type { DashboardMarketRow, DashboardPlanRow, PlanningDashboard } from './api';
import { dashboardHref } from './dashboard-links';
import type { DashboardFilters, DashboardKind } from './dashboard-links';
import { SHARED_PLAN_NOTE, word } from './dashboard-text';
import { FigureTable } from './FigureTable';

interface MarketsTableProps {
  rows: DashboardMarketRow[];
  filters: DashboardFilters;
}

export function MarketsTable({ rows, filters }: MarketsTableProps) {
  return (
    <Card title="By market" subtitle={SHARED_PLAN_NOTE}>
      <FigureTable
        caption="Plans and figures by market; a market opens its Regional dashboard"
        nameTitle="Market"
        extraTitles={['Plans']}
        rows={rows.map((row) => ({
          key: row.team.id,
          name: (
            <Link to={dashboardHref('regional', { ...filters, team: row.team.id, plan: '' })}>
              {row.team.name}
            </Link>
          ),
          extra: [row.planCount],
          figures: row.figures,
        }))}
      />
    </Card>
  );
}

interface PlansTableProps {
  kind: DashboardKind;
  rows: DashboardPlanRow[];
  filters: DashboardFilters;
}

export function PlansTable({ kind, rows, filters }: PlansTableProps) {
  const extraTitles = kind === 'regional' ? ['Status', 'Campaigns'] : ['Status'];
  return (
    <Card title="By plan">
      <FigureTable
        caption="Plans of the dashboard by name"
        nameTitle="Plan"
        extraTitles={extraTitles}
        rows={rows.map((row) => ({
          key: row.plan.id,
          name: <Link to={`/plans/${row.plan.id}`}>{row.plan.name}</Link>,
          extra: [
            word(row.plan.status),
            ...(kind === 'regional'
              ? [
                  <Link
                    key="campaigns"
                    to={dashboardHref('regional', { ...filters, plan: row.plan.id })}
                    aria-label={`Show campaigns of ${row.plan.name}`}
                  >
                    Show campaigns
                  </Link>,
                ]
              : []),
          ],
          figures: row.figures,
          chosen: row.plan.id === filters.plan,
        }))}
      />
    </Card>
  );
}

interface PlanCampaignsTableProps {
  planCampaigns: NonNullable<PlanningDashboard['planCampaigns']>;
}

export function PlanCampaignsTable({ planCampaigns }: PlanCampaignsTableProps) {
  const { plan, items } = planCampaigns;
  return (
    <Card title={`Campaigns of ${plan.name}`}>
      <FigureTable
        caption={`Campaigns of ${plan.name} in the plan's order`}
        nameTitle="Campaign"
        rows={items.map((row) => ({
          key: row.campaign.id,
          name: (
            <Link to={`/plans/${plan.id}/campaigns/${row.campaign.id}`}>{row.campaign.name}</Link>
          ),
          figures: row.figures,
        }))}
      />
    </Card>
  );
}
