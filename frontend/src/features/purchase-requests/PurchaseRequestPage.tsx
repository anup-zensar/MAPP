import { CircleAlert } from 'lucide-react';
import { useParams } from 'react-router';
import { ApiError } from '@/api/errors';
import { HistorySection } from '@/features/activity-log';
import { ApprovalAlert } from '@/features/approvals';
import { useAccess } from '@/session/access';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { usePageTitle } from '@/ui/usePageTitle';
import { usePurchaseRequest } from './api';
import type { PurchaseRequest } from './api';
import { ApprovedVersionCard } from './ApprovedVersionCard';
import { GsapFields } from './GsapFields';
import { GO_TO_PLANS, GO_TO_PURCHASE_REQUESTS } from './page-text';
import { AWAITING_NOTICE, PURCHASE_REQUEST } from './purchase-request-text';
import { PurchaseRequestApprovalSection } from './PurchaseRequestApprovalSection';
import { PurchaseRequestDetailsCard } from './PurchaseRequestDetailsCard';
import { PurchaseRequestHeader } from './PurchaseRequestHeader';
import { PurchaseRequestLinesTable } from './PurchaseRequestLinesTable';
import { RoutingHistory } from './RoutingHistory';
import styles from './PurchaseRequestPage.module.css';

/**
 * A purchase request's page (032 FR-008, FR-009, FR-052; 033 FR-001): header with its status,
 * completeness and actions, the server's notice, details, GSAP data, lines, the approved version
 * while a change waits, approval, routing history and history.
 */
export function PurchaseRequestPage() {
  const { purchaseRequestId = '' } = useParams();
  const purchaseRequest = usePurchaseRequest(purchaseRequestId);
  usePageTitle(purchaseRequest.data?.name ?? PURCHASE_REQUEST);

  if (purchaseRequest.isError) {
    return (
      <LoadFailed error={purchaseRequest.error} onRetry={() => void purchaseRequest.refetch()} />
    );
  }
  if (!purchaseRequest.data) {
    return (
      <>
        <PageHeader breadcrumb={[{ label: 'Plans', to: '/plans' }]} title={PURCHASE_REQUEST} />
        <Card>
          <Skeleton lines={6} />
        </Card>
      </>
    );
  }
  return <Loaded purchaseRequest={purchaseRequest.data} />;
}

function Loaded({ purchaseRequest: pr }: { purchaseRequest: PurchaseRequest }) {
  return (
    <>
      <PurchaseRequestHeader purchaseRequest={pr} />
      <div className={styles.page}>
        {pr.notice && (
          <ApprovalAlert tone={pr.notice === AWAITING_NOTICE ? 'warning' : 'info'}>
            <p>{pr.notice}</p>
          </ApprovalAlert>
        )}
        <div className={styles.columns}>
          <PurchaseRequestDetailsCard purchaseRequest={pr} />
          <GsapFields purchaseRequest={pr} />
        </div>
        <PurchaseRequestLinesTable purchaseRequest={pr} />
        <ApprovedVersionCard purchaseRequest={pr} />
        <div id="approval">
          <PurchaseRequestApprovalSection purchaseRequest={pr} />
        </div>
        <RoutingHistory purchaseRequest={pr} />
        <HistorySection
          recordType="purchaseRequest"
          recordId={pr.id}
          recordName={pr.name}
          level="purchaseRequest"
        />
      </div>
    </>
  );
}

/** Missing or outside the person's access (404), refused (403), or failed: what the server said. */
function LoadFailed({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const { has } = useAccess();
  if (error instanceof ApiError && error.kind === 'forbidden') {
    return <NoAccess message={error.message} />;
  }
  const header = (
    <PageHeader breadcrumb={[{ label: 'Plans', to: '/plans' }]} title={PURCHASE_REQUEST} />
  );
  if (error instanceof ApiError && error.kind === 'notFound') {
    const list = has('purchaseRequests.list');
    return (
      <>
        {header}
        <Card>
          <EmptyState
            icon={<CircleAlert />}
            title={error.message}
            action={
              <LinkButton to={list ? '/purchase-requests' : '/plans'} variant="primary">
                {list ? GO_TO_PURCHASE_REQUESTS : GO_TO_PLANS}
              </LinkButton>
            }
          />
        </Card>
      </>
    );
  }
  return (
    <>
      {header}
      <Card>
        <ErrorState error={error} onRetry={onRetry} />
      </Card>
    </>
  );
}
