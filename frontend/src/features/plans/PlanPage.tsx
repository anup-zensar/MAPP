import { useEffect, useRef } from 'react';
import { useNavigate, useOutlet, useParams, useSearchParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button, ErrorState, Skeleton, Tabs } from '@/components';
import type { ActionsMenuItem, TabItem } from '@/components';
import { HistorySection } from '@/features/activity-log/HistorySection';
import { showHistory } from '@/features/activity-log/showHistory';
import { useCampaignTreeContent } from '@/features/campaigns/useCampaignTreeContent';
import { usePageTitle } from '@/lib/usePageTitle';
import { PlanDetails } from './PlanDetails';
import { PlanGoals } from './PlanGoals';
import { PlanHeader } from './PlanHeader';
import { PlanNotFound } from './PlanNotFound';
import styles from './PlanPage.module.css';
import { usePlan } from './plansApi';
import type { Plan } from './plansApi';
import { PlanSummaryCards } from './PlanSummaryCards';
import { PlanTree } from './tree/PlanTree';
import { usePlanActions } from './usePlanActions';

/** The plan page's tabs (design system §2.4 B): the tree is the working view. */
export type PlanTab = 'tree' | 'details' | 'history';

const TABS: TabItem<PlanTab>[] = [
  { value: 'tree', label: 'Plan tree' },
  { value: 'details', label: 'Details' },
  { value: 'history', label: 'History' },
];

/** The chosen tab, kept in the address (?tab=details); the tree when none or an unknown one. */
function tabOf(search: URLSearchParams): PlanTab {
  const tab = search.get('tab');
  return tab === 'details' || tab === 'history' ? tab : 'tree';
}

/**
 * A plan's page (US3, FR-011 – FR-017), the layout route of /plans/:planId, on the record template
 * (design system §2.4 B): the header, four summary cards, then the tabs **Plan tree · Details ·
 * History** with the chosen one in the address. Feature 005 fills the tree through
 * `useCampaignTreeContent` and shows its side panels in the `<Outlet />` (child routes); the
 * browser tab names the plan unless a panel on top names itself.
 */
export function PlanPage() {
  const { planId = '' } = useParams();
  const navigate = useNavigate();
  const plan = usePlan(planId);
  const campaigns = useCampaignTreeContent(planId);
  const panel = useOutlet();
  const [search, setSearch] = useSearchParams();
  const tab = tabOf(search);
  // View history: once the History tab is shown, its heading takes the focus (feature 011).
  const focusHistory = useRef(false);
  usePageTitle(panel ? null : (plan.data?.name ?? 'Plan'));
  const actions = usePlanActions({ onDeleted: () => void navigate('/plans') });

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
    const moreItems: ActionsMenuItem[] = [
      actions.copyItem(data),
      // Opens the History tab and focuses its heading (feature 011, FR-011).
      { label: 'View history', onSelect: viewHistory },
      actions.deleteItem(data),
    ];
    return (
      <>
        <PlanHeader
          plan={data}
          moreItems={moreItems}
          statusAction={
            data.status === 'draft' ? (
              <Button onClick={() => void actions.publish(data)}>Publish</Button>
            ) : (
              <Button onClick={() => actions.unpublish(data)}>Unpublish</Button>
            )
          }
        />
        <div className={styles.body}>
          <PlanSummaryCards plan={data} />
          <Tabs label="Plan" tabs={TABS} value={tab} onChange={chooseTab}>
            {tabContent(data)}
          </Tabs>
        </div>
      </>
    );
  }

  return (
    <div className={styles.page}>
      {content()}
      {actions.dialogs}
      {campaigns.dialogs}
      {panel}
    </div>
  );
}

/** The page's shape while the plan loads: header, summary cards and tree. */
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
