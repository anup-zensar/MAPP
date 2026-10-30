import { useNavigate, useParams, useSearchParams } from 'react-router';
import { useAccess } from '@/session/access';
import type { BreadcrumbItem } from '@/ui/Breadcrumb/Breadcrumb';
import { purchaseRequestHref } from '@/ui/record-links';
import { useToast } from '@/ui/Toast/useToast';
import { usePageTitle } from '@/ui/usePageTitle';
import { usePurchaseRequest, usePurchaseRequestConfiguration, useSavePurchaseRequest } from './api';
import type { PurchaseRequest } from './api';
import { fromPurchaseRequest } from './form-state';
import { savedToast } from './form-text';
import { FormPageFailed, FormPageLoading, FormPageUnavailable } from './FormPageStates';
import { purchaseRequestBreadcrumb } from './page-links';
import { CLOSED_NO_CHANGE, EDIT_PURCHASE_REQUEST } from './purchase-request-text';
import { PurchaseRequestForm } from './PurchaseRequestForm';
import type { PurchaseRequestFormOptions } from './PurchaseRequestForm';
import { useFormOptions } from './useFormOptions';

const BACK = 'Go to the purchase request';
const NOT_YOURS = "Changing this purchase request isn't part of your role.";

/** Edit purchase request (032 US5, FR-016, FR-023, FR-036, FR-039, FR-053, FR-054). */
export function PurchaseRequestEditPage() {
  const { purchaseRequestId = '' } = useParams();
  const pr = usePurchaseRequest(purchaseRequestId);
  const configuration = usePurchaseRequestConfiguration();
  const { options, failed } = useFormOptions(pr.data);
  usePageTitle(EDIT_PURCHASE_REQUEST);
  const loadFailed = pr.isError ? pr : failed;

  if (loadFailed) {
    return (
      <FormPageFailed
        title={EDIT_PURCHASE_REQUEST}
        error={loadFailed.error}
        onRetry={() => void loadFailed.refetch()}
      />
    );
  }
  if (!pr.data || !options) {
    return <FormPageLoading title={EDIT_PURCHASE_REQUEST} />;
  }
  const linesOnApproved = configuration.data?.linesOnApprovedPurchaseRequests;
  return (
    <Loaded
      pr={pr.data}
      options={{ ...options, linesOnApproved }}
      onReload={() => void pr.refetch()}
    />
  );
}

function breadcrumbOf(pr: PurchaseRequest, seesList: boolean): BreadcrumbItem[] {
  const path = purchaseRequestBreadcrumb(pr, seesList).slice(0, -1);
  return [
    ...path,
    { label: pr.name, to: purchaseRequestHref(pr.id) },
    { label: EDIT_PURCHASE_REQUEST },
  ];
}

interface LoadedProps {
  pr: PurchaseRequest;
  options: PurchaseRequestFormOptions;
  onReload: () => void;
}

function Loaded({ pr, options, onReload }: LoadedProps) {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const { show } = useToast();
  const save = useSavePurchaseRequest(pr.id);
  const href = purchaseRequestHref(pr.id);
  const { has } = useAccess();
  const breadcrumb = breadcrumbOf(pr, has('purchaseRequests.list'));

  if (pr.status === 'closed' || !pr.access.edit.available) {
    const reason = pr.status === 'closed' ? CLOSED_NO_CHANGE : pr.access.edit.reason || NOT_YOURS;
    return (
      <FormPageUnavailable
        title={EDIT_PURCHASE_REQUEST}
        breadcrumb={breadcrumb}
        reason={reason}
        backTo={href}
        backLabel={BACK}
      />
    );
  }
  return (
    <PurchaseRequestForm
      title={EDIT_PURCHASE_REQUEST}
      breadcrumb={breadcrumb}
      initial={fromPurchaseRequest(pr)}
      stored={pr}
      options={options}
      optionsQuery={{ purchaseRequestId: pr.id }}
      focusField={params.get('field')}
      onSave={(input) => save.mutateAsync({ ...input, version: pr.version })}
      onSaved={(saved) => {
        show({ message: savedToast(saved.name) });
        void navigate(href);
      }}
      onReload={onReload}
      onCancel={() => void navigate(href)}
    />
  );
}
