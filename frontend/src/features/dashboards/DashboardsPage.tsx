import { LayoutDashboard } from 'lucide-react';
import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { useFormatters } from '@/format/useFormatters';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { Tabs } from '@/ui/Tabs/Tabs';
import { usePageTitle } from '@/ui/usePageTitle';
import { downloadDashboard, useDashboard } from './api';
import type { PlanningDashboard } from './api';
import { ApprovalsPanel } from './ApprovalsPanel';
import { CampaignLists } from './CampaignLists';
import { CompletenessPanel } from './CompletenessPanel';
import { correctedFilters, dashboardHref, writeFilters } from './dashboard-links';
import type { DashboardFilters as Filters, DashboardKind } from './dashboard-links';
import { emptyText, financeCoverage } from './dashboard-text';
import { DashboardFilters } from './DashboardFilters';
import { MarketsTable, PlanCampaignsTable, PlansTable } from './DashboardTables';
import { ExportMenu } from './ExportMenu';
import { FigureCards } from './FigureCards';
import { GlobalBudgetsPanel } from './GlobalBudgetsPanel';
import { MonthlyChart } from './MonthlyChart';
import { PurchaseRequestsPanel } from './PurchaseRequestsPanel';
import { readFilters } from './schemas';
import styles from './Dashboards.module.css';

const TITLES: Record<DashboardKind, string> = {
  executive: 'Executive dashboard',
  regional: 'Regional dashboard',
};

const DESCRIPTIONS: Record<DashboardKind, string> = {
  executive: 'Budget, planned cost and spend of every market you can see.',
  regional: 'Budget, planned cost and spend of one market, plan by plan.',
};

interface DashboardsPageProps {
  kind: DashboardKind;
}

export function DashboardsPage({ kind }: DashboardsPageProps) {
  usePageTitle(TITLES[kind]);
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const filters = readFilters(params);
  const dashboard = useDashboard(kind, filters);
  const forbidden = dashboard.error instanceof ApiError && dashboard.error.kind === 'forbidden';
  const corrected =
    dashboard.data && !dashboard.isPlaceholderData
      ? correctedFilters(filters, dashboard.data.filters)
      : null;
  const correctedSearch = corrected ? writeFilters(corrected).toString() : null;

  // The address shows the filters the dashboard was made with.
  useEffect(() => {
    if (correctedSearch !== null) {
      setParams(correctedSearch, { replace: true });
    }
  }, [correctedSearch, setParams]);

  if (forbidden) {
    return <NoAccess />;
  }

  function change(patch: Partial<Filters>) {
    setParams(writeFilters({ ...filters, ...patch }));
  }

  return (
    <>
      <PageHeader
        eyebrow="Dashboards"
        title={TITLES[kind]}
        description={DESCRIPTIONS[kind]}
        actions={
          dashboard.data?.canExport && (
            <ExportMenu download={(format) => downloadDashboard(kind, filters, format)} />
          )
        }
      />
      <Tabs
        label="Dashboards"
        tabs={[
          { value: 'executive', label: 'Executive' },
          { value: 'regional', label: 'Regional' },
        ]}
        value={kind}
        onChange={(next) => void navigate(dashboardHref(next, { ...filters, team: '', plan: '' }))}
      >
        <div className={styles.sections} aria-busy={dashboard.isFetching || undefined}>
          {dashboard.data ? (
            <Dashboard kind={kind} dashboard={dashboard.data} filters={filters} onChange={change} />
          ) : dashboard.isError ? (
            <ErrorState error={dashboard.error} onRetry={() => void dashboard.refetch()} />
          ) : (
            <div aria-label="Loading the dashboard">
              <Skeleton lines={8} height="24px" />
            </div>
          )}
        </div>
      </Tabs>
    </>
  );
}

interface DashboardProps {
  kind: DashboardKind;
  dashboard: PlanningDashboard;
  filters: Filters;
  onChange: (patch: Partial<Filters>) => void;
}

function Dashboard({ kind, dashboard, filters, onChange }: DashboardProps) {
  const { dateTime } = useFormatters();
  const coverage = financeCoverage(dashboard.finance.planCount, dashboard.planCount);
  const filtersBar = <DashboardFilters kind={kind} dashboard={dashboard} onChange={onChange} />;
  if (dashboard.planCount === 0) {
    return (
      <>
        {filtersBar}
        <div className={styles.onGrey}>
          <EmptyState
            icon={<LayoutDashboard />}
            title={emptyText(dashboard.filters.year, dashboard.team?.name ?? null)}
            text="Choose another year, market or plan type."
          />
        </div>
      </>
    );
  }
  return (
    <>
      {filtersBar}
      {coverage && <p className={styles.note}>{coverage}</p>}
      {dashboard.financeAsOf && (
        <p className={styles.note}>GSAP data as of {dateTime(dashboard.financeAsOf)}</p>
      )}
      <FigureCards figures={dashboard.totals} />
      {kind === 'executive' && <MarketsTable rows={dashboard.markets} filters={filters} />}
      <PlansTable kind={kind} rows={dashboard.plans} filters={filters} />
      {dashboard.planCampaigns && <PlanCampaignsTable planCampaigns={dashboard.planCampaigns} />}
      <MonthlyChart months={dashboard.months} />
      <div className={styles.grid}>
        <CampaignLists overBudget={dashboard.overBudget} leftToPlan={dashboard.leftToPlan} />
        <ApprovalsPanel approvals={dashboard.approvals} />
        <PurchaseRequestsPanel purchaseRequests={dashboard.purchaseRequests} />
        <CompletenessPanel levels={dashboard.completeness} />
        {dashboard.globalBudgets && <GlobalBudgetsPanel budgets={dashboard.globalBudgets} />}
      </div>
    </>
  );
}
