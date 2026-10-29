import { useEffect, useRef } from 'react';
import { useNavigate, useOutlet, useParams, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { Tabs } from '@/ui/Tabs/Tabs';
import { ViewOnlyNote } from '@/ui/ViewOnlyNote/ViewOnlyNote';
import { ViewOnlyArea } from '@/ui/ViewOnlyNote/view-only-area';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import type { TabItem } from '@/ui/Tabs/Tabs';
import { HistorySection, showHistory } from '@/features/activity-log';
import { usePageTitle } from '@/ui/usePageTitle';
import { allows, moreItemFor } from '@/session/gate-action';
import { useWorkbookMenu } from '@/features/workbook';
import { BudgetByMonthCard } from './phasing/BudgetByMonthCard';
import { useCampaignTreeContent } from './campaigns/useCampaignTreeContent';
import { PlanDetails } from './PlanDetails';
import { PlanGoals } from './PlanGoals';
import { PlanHeader } from './PlanHeader';
import { PlannedCostByCard } from './PlannedCostByCard';
import { PlanNotFound } from './PlanNotFound';
import styles from './PlanPage.module.css';
import { usePlan } from './api';
import type { Plan } from './api';
import { changesNothingIn } from './plan-access';
import { planCurrencyOf } from './plan-currency';
import { PlanSummaryCards } from './PlanSummaryCards';
import { RecordFinancials } from './RecordFinancials';
import { planTabOf } from './schemas';
import type { PlanTab } from './schemas';
import { PlanTree } from './tree/PlanTree';
import { usePlanActions } from './usePlanActions';
import { usePlanExport } from './usePlanExport';

const TABS: TabItem<PlanTab>[] = [
  { value: 'tree', label: 'Plan tree' },
  { value: 'details', label: 'Details' },
  { value: 'history', label: 'History' },
];

// The layout route of /plans/:planId: header, summary cards, Financials, Budget by month and Planned
// cost by, then the tabs Plan tree · Details · History. Campaign side panels show in the outlet.
export function PlanPage() {
  const { planId = '' } = useParams();
  const navigate = useNavigate();
  const plan = usePlan(planId);
  const campaigns = useCampaignTreeContent(planId, plan.data?.access?.content.addCampaign);
  const panel = useOutlet();
  const [search, setSearch] = useSearchParams();
  const tab = planTabOf(search);
  // View history: once the History tab is shown, its heading takes the focus (feature 011).
  const focusHistory = useRef(false);
  usePageTitle(panel ? null : (plan.data?.name ?? 'Plan'));
  const actions = usePlanActions({ onDeleted: () => void navigate('/plans') });
  const exportItems = usePlanExport(planId);
  const workbookItems = useWorkbookMenu(planId, allows(plan.data?.access?.content.editActivities));
  const viewOnly = changesNothingIn(plan.data?.access);

  useEffect(() => {
    if (tab === 'history' && focusHistory.current) {
      focusHistory.current = false;
      showHistory();
    }
  }, [tab]);

  function chooseTab(next: PlanTab) {
    setSearch(
      (current) => {
        const params = new URLSearchParams(current);
        if (next === 'tree') {
          params.delete('tab');
        } else {
          params.set('tab', next);
        }
        return params;
      },
      { replace: true },
    );
  }

  function viewHistory() {
    if (tab === 'history') {
      showHistory();
    } else {
      focusHistory.current = true;
      chooseTab('history');
    }
  }

  function tabContent(data: Plan) {
    switch (tab) {
      case 'details':
        return (
          <div className={styles.columns}>
            <PlanDetails plan={data} />
            <PlanGoals plan={data} />
          </div>
        );
      case 'history':
        return (
          <HistorySection
            recordType="plan"
            recordId={data.id}
            recordName={data.name}
            level="plan"
          />
        );
      default:
        return (
          <PlanTree
            plan={data}
            childNodes={campaigns.nodes}
            toolbarActions={campaigns.toolbarActions}
            footer={campaigns.footer}
            planRowActions={campaigns.planRowActions}
            emptyAction={campaigns.emptyAction}
            isLoading={campaigns.isLoading}
            error={campaigns.error}
            onRetry={campaigns.refetch}
          />
        );
    }
  }

  function content() {
    if (plan.isPending) {
      return <PlanPageSkeleton />;
    }
    if (plan.isError) {
      if (plan.error instanceof ApiError && plan.error.kind === 'notFound') {
        return <PlanNotFound />;
      }
      return <ErrorState error={plan.error} onRetry={() => void plan.refetch()} />;
    }
    const data = plan.data;
    const status: ActionsMenuItem =
      data.status === 'draft'
        ? { label: 'Publish', onSelect: () => void actions.publish(data) }
        : { label: 'Unpublish', onSelect: () => actions.unpublish(data) };
    const moreItems: ActionsMenuItem[] = [
      ...moreItemFor(
        {
          label: 'Edit',
          onSelect: () => void navigate(`/plans/${encodeURIComponent(data.id)}/edit`),
        },
        data.access?.edit,
      ),
      ...moreItemFor(status, data.access?.publish),
      ...actions.copyItem(data),
      { label: 'View history', onSelect: viewHistory },
      ...workbookItems,
      ...exportItems,
      ...actions.deleteItem(data),
    ];
    return (
      <>
        <PlanHeader
          plan={data}
          moreItems={moreItems}
          statusAction={
            allows(data.access?.publish) && (
              <Button onClick={status.onSelect}>{status.label}</Button>
            )
          }
        />
        <div className={styles.body}>
          {viewOnly && <ViewOnlyNote area="plan" />}
          <PlanSummaryCards plan={data} />
          {/* D-305: Financials, Budget by month and Planned cost by, open, above the tabs. */}
          <RecordFinancials recordId={data.id} planId={data.id} variant="page" />
          <BudgetByMonthCard
            level="plan"
            planId={data.id}
            record={data}
            year={data.year}
            currency={planCurrencyOf(data)}
            access={data.access?.edit}
          />
          <PlannedCostByCard plan={data} />
          <Tabs label="Plan" tabs={TABS} value={tab} onChange={chooseTab}>
            {tabContent(data)}
          </Tabs>
        </div>
      </>
    );
  }

  return (
    <ViewOnlyArea active={viewOnly}>
      <div className={styles.page}>
        {content()}
        {actions.dialogs}
        {campaigns.dialogs}
        {panel}
      </div>
    </ViewOnlyArea>
  );
}

function PlanPageSkeleton() {
  return (
    <div className={styles.body} aria-busy="true" aria-label="Loading the plan">
      <div className={styles.skeletonHeader}>
        <Skeleton width="180px" />
        <Skeleton width="50%" height="28px" />
        <Skeleton width="70%" />
      </div>
      <div className={styles.summary}>
        {[0, 1, 2, 3].map((card) => (
          <div key={card} className={styles.skeletonCard}>
            <Skeleton width="60%" />
            <Skeleton width="40%" height="24px" />
          </div>
        ))}
      </div>
      <div className={styles.skeletonCard}>
        <Skeleton lines={4} />
      </div>
    </div>
  );
}
