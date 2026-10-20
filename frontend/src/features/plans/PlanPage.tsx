import { useNavigate, useOutlet, useParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button, ErrorState, Skeleton } from '@/components';
import type { ActionsMenuItem } from '@/components';
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
import { PlanSummaryCards } from './PlanSummaryCards';
import { PlanTree } from './tree/PlanTree';
import { usePlanActions } from './usePlanActions';

/**
 * A plan's page (US3, FR-011 – FR-017), the layout route of /plans/:planId: the header, the
 * summary cards, the plan tree, Details and Goals, then the History (feature 011). Feature 005 fills the tree through
 * `useCampaignTreeContent` and shows its side panels in the `<Outlet />` (child routes); the tab
 * names the plan unless a panel on top names itself.
 */
export function PlanPage() {
  const { planId = '' } = useParams();
  const navigate = useNavigate();
  const plan = usePlan(planId);
  const campaigns = useCampaignTreeContent(planId);
  const panel = useOutlet();
  usePageTitle(panel ? null : (plan.data?.name ?? 'Plan'));
  const actions = usePlanActions({ onDeleted: () => void navigate('/plans') });

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
      // Brings the History section below into view and focuses it (feature 011, FR-011).
      { label: 'View history', onSelect: () => showHistory() },
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
        <PlanSummaryCards plan={data} />
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
        <div className={styles.columns}>
          <PlanDetails plan={data} />
          <PlanGoals plan={data} />
        </div>
        <HistorySection recordType="plan" recordId={data.id} recordName={data.name} level="plan" />
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
    <div className={styles.page} aria-busy="true" aria-label="Loading the plan">
      <div className={styles.skeletonHeader}>
        <Skeleton width="180px" />
        <Skeleton width="50%" height="28px" />
        <Skeleton width="70%" />
      </div>
      <div className={styles.summary}>
        {[0, 1, 2, 3, 4].map((card) => (
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
