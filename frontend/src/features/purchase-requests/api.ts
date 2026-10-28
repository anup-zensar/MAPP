/**
 * What the purchase-request screens ask the API (032 data-model §5; 033). A purchase request's
 * figures show on line items, activities, approvals, Home and Financials, whose keys this feature
 * does not own: so after every change every query but the session is asked again.
 */
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import { downloadFromApi } from '@/api/download';
import type { components, operations } from '@/api/generated/schema';

type Schemas = components['schemas'];
export type PurchaseRequest = Schemas['PurchaseRequest'];
export type PurchaseRequestLine = Schemas['PurchaseRequestLine'];
export type PurchaseRequestStatus = Schemas['PurchaseRequestStatus'];
export type PurchaseRequestInput = Schemas['PurchaseRequestInput'];
export type PurchaseRequestLineInput = Schemas['PurchaseRequestLineInput'];
export type PurchaseRequestDefaults = Schemas['PurchaseRequestDefaults'];
export type PurchaseRequestLineItemRef = Schemas['PurchaseRequestLineItemRef'];
export type PurchaseRequestLineItemOption = Schemas['PurchaseRequestLineItemOption'];
export type PurchaseRequestListPage = Schemas['PurchaseRequestListPage'];
export type PurchaseRequestRow = Schemas['PurchaseRequestRow'];
export type PurchaseRequestRowLine = Schemas['PurchaseRequestRowLine'];
export type PurchaseRequestVendorGroup = Schemas['PurchaseRequestVendorGroup'];
export type PurchaseRequestRouting = Schemas['PurchaseRequestRouting'];
export type PurchaseRequestRoutingType = Schemas['PurchaseRequestRoutingType'];
export type PurchaseRequestVersion = Schemas['PurchaseRequestVersion'];
export type PurchaseRequestProblem = Schemas['PurchaseRequestProblem'];
export type PurchaseRequestAccess = Schemas['PurchaseRequestAccess'];
export type PurchaseRequestApproval = Schemas['PurchaseRequestApproval'];
export type PurchaseRequestApprovalSection = Schemas['PurchaseRequestApprovalSection'];
export type PurchaseRequestSubmission = Schemas['PurchaseRequestSubmission'];
export type PurchaseRequestClosePreview = Schemas['PurchaseRequestClosePreview'];
export type PurchaseRequestConfiguration = Schemas['PurchaseRequestConfiguration'];
export type PurchaseRequestConfigurationInput = Schemas['PurchaseRequestConfigurationInput'];
export type LinesOnApproved = Schemas['LinesOnApprovedPurchaseRequests'];
export type LineItemPurchaseRequests = Schemas['LineItemPurchaseRequests'];
export type ActivityPurchaseRequests = Schemas['ActivityPurchaseRequests'];
export type GsapSource = Schemas['GsapSource'];
export type Money = Schemas['Money'];

export type PurchaseRequestListQuery = NonNullable<
  operations['listPurchaseRequests']['parameters']['query']
>;
export type LineItemOptionQuery = NonNullable<
  operations['listPurchaseRequestLineItemOptions']['parameters']['query']
>;
export type FormFormat = 'xlsx' | 'csv';

const ROOT = 'purchase-requests';

export const purchaseRequestKeys = {
  all: [ROOT],
  list: (query: PurchaseRequestListQuery) => [ROOT, 'list', query],
  detail: (id: string) => [ROOT, 'detail', id],
  approval: (id: string) => [ROOT, 'approval', id],
  submission: (id: string) => [ROOT, 'submission', id],
  lineItem: (id: string) => [ROOT, 'line-item', id],
  activity: (id: string) => [ROOT, 'activity', id],
  options: (query: LineItemOptionQuery) => [ROOT, 'options', query],
  defaults: (lineItemId: string) => [ROOT, 'defaults', lineItemId],
  closePreview: (id: string) => [ROOT, 'close-preview', id],
  configuration: [ROOT, 'configuration'],
};

function path(purchaseRequestId: string) {
  return { params: { path: { purchaseRequestId } } };
}

export function usePurchaseRequestList(query: PurchaseRequestListQuery) {
  return useQuery({
    queryKey: purchaseRequestKeys.list(query),
    queryFn: () => call(api.GET('/api/v1/purchase-requests', { params: { query } })),
    placeholderData: keepPreviousData,
  });
}

export function usePurchaseRequest(id: string) {
  return useQuery({
    queryKey: purchaseRequestKeys.detail(id),
    queryFn: () => call(api.GET('/api/v1/purchase-requests/{purchaseRequestId}', path(id))),
  });
}

export function usePurchaseRequestDefaults(lineItemId: string) {
  return useQuery({
    queryKey: purchaseRequestKeys.defaults(lineItemId),
    queryFn: () =>
      call(
        api.GET('/api/v1/purchase-requests/new-defaults', { params: { query: { lineItemId } } }),
      ),
    enabled: lineItemId !== '',
    staleTime: 0,
  });
}

export function useLineItemOptions(query: LineItemOptionQuery, enabled = true) {
  return useQuery({
    queryKey: purchaseRequestKeys.options(query),
    queryFn: () =>
      call(api.GET('/api/v1/purchase-requests/line-item-options', { params: { query } })),
    enabled,
    placeholderData: keepPreviousData,
  });
}

export function usePurchaseRequestApproval(id: string) {
  return useQuery({
    queryKey: purchaseRequestKeys.approval(id),
    queryFn: () =>
      call(api.GET('/api/v1/purchase-requests/{purchaseRequestId}/approval', path(id))),
  });
}

/** What Submit or Re-submit would do now; always asked afresh while its dialog is open. */
export function usePurchaseRequestSubmission(id: string, enabled: boolean) {
  return useQuery({
    queryKey: purchaseRequestKeys.submission(id),
    queryFn: () =>
      call(api.GET('/api/v1/purchase-requests/{purchaseRequestId}/approval/submission', path(id))),
    enabled,
    staleTime: 0,
    gcTime: 0,
  });
}

export function useClosePreview(id: string, enabled: boolean) {
  return useQuery({
    queryKey: purchaseRequestKeys.closePreview(id),
    queryFn: () =>
      call(api.GET('/api/v1/purchase-requests/{purchaseRequestId}/close-preview', path(id))),
    enabled,
    staleTime: 0,
    gcTime: 0,
  });
}

export function useLineItemPurchaseRequests(lineItemId: string) {
  return useQuery({
    queryKey: purchaseRequestKeys.lineItem(lineItemId),
    queryFn: () =>
      call(
        api.GET('/api/v1/line-items/{lineItemId}/purchase-requests', {
          params: { path: { lineItemId } },
        }),
      ),
  });
}

export function useActivityPurchaseRequests(activityId: string) {
  return useQuery({
    queryKey: purchaseRequestKeys.activity(activityId),
    queryFn: () =>
      call(
        api.GET('/api/v1/activities/{activityId}/purchase-requests', {
          params: { path: { activityId } },
        }),
      ),
  });
}

export type RoundKind = Schemas['PurchaseRequestRoundKind'];
type Workflow = Schemas['ApprovalWorkflow'];

function kindOf(workflow: Workflow): string {
  return workflow.objectType;
}

/** The active workflows of one kind, for the PR's two workflow pickers (FR-005). */
export function useWorkflowOptions(kind: RoundKind) {
  return useQuery({
    queryKey: ['approval-workflows'],
    queryFn: () => call(api.GET('/api/v1/approval-workflows')),
    select: (page) => page.items.filter((workflow) => workflow.active && kindOf(workflow) === kind),
  });
}

export function usePurchaseRequestConfiguration() {
  return useQuery({
    queryKey: purchaseRequestKeys.configuration,
    queryFn: () => call(api.GET('/api/v1/purchase-request-configuration')),
  });
}

/**
 * A mutation whose every answer, a refusal too, refreshes every query but the session and the new-PR
 * defaults: a saved PR can use up its line item, and refreshing them would replace the page being left.
 */
function useChange<I, O>(mutationFn: (input: I) => Promise<O>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSettled: () =>
      queryClient.invalidateQueries({
        predicate: ({ queryKey }) => queryKey[0] !== 'session' && !isDefaultsKey(queryKey),
      }),
  });
}

function isDefaultsKey(queryKey: readonly unknown[]): boolean {
  return queryKey[0] === ROOT && queryKey[1] === 'defaults';
}

export function useCreatePurchaseRequest() {
  return useChange((body: PurchaseRequestInput) =>
    call(api.POST('/api/v1/purchase-requests', { body })),
  );
}

export function useSavePurchaseRequest(id: string) {
  return useChange((body: PurchaseRequestInput) =>
    call(api.PUT('/api/v1/purchase-requests/{purchaseRequestId}', { ...path(id), body })),
  );
}

export function deletePurchaseRequest(id: string) {
  return call(api.DELETE('/api/v1/purchase-requests/{purchaseRequestId}', path(id)));
}

export function useClosePurchaseRequest(id: string) {
  return useChange((version: number) =>
    call(
      api.POST('/api/v1/purchase-requests/{purchaseRequestId}/close', {
        ...path(id),
        body: { version },
      }),
    ),
  );
}

export function useResendPurchaseRequestForm(id: string) {
  return useChange((version: number) =>
    call(
      api.POST('/api/v1/purchase-requests/{purchaseRequestId}/resend', {
        ...path(id),
        body: { version },
      }),
    ),
  );
}

/** Saves the PR form of one routing as a file and returns its name. */
export function downloadPurchaseRequestForm(id: string, routingId: string, format: FormFormat) {
  return downloadFromApi(
    api.GET('/api/v1/purchase-requests/{purchaseRequestId}/routings/{routingId}/form', {
      params: { path: { purchaseRequestId: id, routingId }, query: { format } },
      parseAs: 'blob',
    }),
    `${id} PR form.${format}`,
  );
}

export function useSubmitPurchaseRequest(id: string) {
  return useChange(({ version, comment }: { version: number; comment: string }) =>
    call(
      api.POST('/api/v1/purchase-requests/{purchaseRequestId}/approval/submit', {
        ...path(id),
        body: comment ? { version, comment } : { version },
      }),
    ),
  );
}

export function useRecallPurchaseRequest(id: string) {
  return useChange(({ roundId, reason }: { roundId: string; reason: string }) =>
    call(
      api.POST('/api/v1/purchase-requests/{purchaseRequestId}/approval/recall', {
        ...path(id),
        body: reason ? { roundId, reason } : { roundId },
      }),
    ),
  );
}

export function useSavePurchaseRequestConfiguration() {
  return useChange((body: PurchaseRequestConfigurationInput) =>
    call(api.PUT('/api/v1/purchase-request-configuration', { body })),
  );
}

/** 033: the simulated GSAP creates the GSAP PR and the PO, or tops up the PO. */
export function useCreateGsapOrder(id: string) {
  return useChange((version: number) =>
    call(
      api.POST('/api/v1/simulated-gsap/purchase-requests/{purchaseRequestId}/purchase-order', {
        ...path(id),
        body: { version },
      }),
    ),
  );
}
