import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { invalidateAllButSession } from './api';

export type PurchaseRequestApproval = components['schemas']['PurchaseRequestApproval'];
export type PurchaseRequestApprovalSection =
  components['schemas']['PurchaseRequestApprovalSection'];
export type PurchaseRequestApprovalPreview =
  components['schemas']['PurchaseRequestApprovalPreview'];

/** The approvals page's preview of a purchase request (032 FR-029). */
export function usePurchaseRequestPreview(purchaseRequestId: string) {
  return useQuery({
    queryKey: ['approval-preview', purchaseRequestId],
    queryFn: () =>
      call(
        api.GET('/api/v1/approvals/purchase-requests/{purchaseRequestId}', {
          params: { path: { purchaseRequestId } },
        }),
      ),
  });
}

/** A decision on the level the person saw, in the round they saw. */
export interface PurchaseRequestDecisionInput {
  mode: 'approve' | 'decline';
  purchaseRequestId: string;
  roundId: string;
  level: number;
  comment: string;
}

export function usePurchaseRequestDecision() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      mode,
      purchaseRequestId,
      roundId,
      level,
      comment,
    }: PurchaseRequestDecisionInput) => {
      const options = {
        params: { path: { purchaseRequestId } },
        body: comment ? { roundId, level, comment } : { roundId, level },
      };
      return call(
        mode === 'approve'
          ? api.POST('/api/v1/purchase-requests/{purchaseRequestId}/approval/approve', options)
          : api.POST('/api/v1/purchase-requests/{purchaseRequestId}/approval/decline', options),
      );
    },
    onSettled: () => invalidateAllButSession(queryClient),
  });
}
