/**
 * What feature 008's approval parts ask the API (plan, "Server state"): the activity's approval,
 * the submission check, the approvals page's rows and preview, and the approval actions.
 *
 * An approval action changes an activity's status, which shows in plans, campaigns, initiatives,
 * activity pages, lists and counts of several features, whose query keys this feature does not
 * own: so after every action (also a refused one) every query but the session is asked again.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { useMonthRange } from './useMonthRange';

export type ActivityApproval = components['schemas']['ActivityApproval'];
export type ApprovalSubmission = components['schemas']['ApprovalSubmission'];
export type ApprovalRound = components['schemas']['ApprovalRound'];
export type ApprovalRow = components['schemas']['ApprovalRow'];
export type ApprovalPreview = components['schemas']['ApprovalPreview'];
export type ApprovalPerson = components['schemas']['ApprovalPerson'];
export type ApprovalMonitoring = components['schemas']['ApprovalMonitoring'];
export type EscalationRunResult = components['schemas']['EscalationRunResult'];

/** The three tabs of the approvals page. */
export type ApprovalTab = 'waiting' | 'open' | 'decided';

/** The query key of an activity's approval (its Approval section and status button share it). */
export function approvalKey(activityId: string) {
  return ['approval', activityId];
}

/** Asks every query again except the session's (plan, "Server state"). */
export function invalidateAllButSession(queryClient: QueryClient): Promise<void> {
  return queryClient.invalidateQueries({ predicate: (query) => query.queryKey[0] !== 'session' });
}

/** The activity's approval: chain, history, latest decline, viewer and available actions. */
export function useActivityApproval(activityId: string, enabled = true) {
  return useQuery({
    queryKey: approvalKey(activityId),
    queryFn: () =>
      call(
        api.GET('/api/v1/activities/{activityId}/approval', { params: { path: { activityId } } }),
      ),
    enabled,
  });
}

/** What Submit would do now: every problem, or the levels to confirm. Always asked afresh. */
export function fetchSubmission(
  queryClient: QueryClient,
  activityId: string,
): Promise<ApprovalSubmission> {
  return queryClient.fetchQuery({
    queryKey: ['approval-submission', activityId],
    queryFn: () =>
      call(
        api.GET('/api/v1/activities/{activityId}/approval/submission', {
          params: { path: { activityId } },
        }),
      ),
    staleTime: 0,
  });
}

/** One page of rows of the approvals page; `enabled` false waits (the page still picks its tab). */
export function useApprovalRows(tab: ApprovalTab, enabled = true) {
  return useQuery({
    queryKey: ['approvals', tab],
    queryFn: () => call(api.GET('/api/v1/approvals', { params: { query: { tab } } })),
    enabled,
  });
}

/** The preview side panel of the approvals page. */
export function useApprovalPreview(activityId: string) {
  return useQuery({
    queryKey: ['approval-preview', activityId],
    queryFn: () =>
      call(
        api.GET('/api/v1/approvals/activities/{activityId}', { params: { path: { activityId } } }),
      ),
  });
}

/** Submit (or re-submit) with the activity version the person saw and an optional comment. */
export interface SubmitInput {
  activityId: string;
  version: number;
  comment: string;
}

export function useSubmitActivity() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ activityId, version, comment }: SubmitInput) =>
      call(
        api.POST('/api/v1/activities/{activityId}/approval/submit', {
          params: { path: { activityId } },
          body: comment ? { version, comment } : { version },
        }),
      ),
    onSettled: () => invalidateAllButSession(queryClient),
  });
}

/** A decision on the level the person saw, in the round they saw. */
export interface DecisionInput {
  mode: 'approve' | 'decline';
  activityId: string;
  roundId: string;
  level: number;
  comment: string;
}

export function useDecide() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ mode, activityId, roundId, level, comment }: DecisionInput) => {
      const body = comment ? { roundId, level, comment } : { roundId, level };
      const params = { path: { activityId } };
      return call(
        mode === 'approve'
          ? api.POST('/api/v1/activities/{activityId}/approval/approve', { params, body })
          : api.POST('/api/v1/activities/{activityId}/approval/decline', { params, body }),
      );
    },
    onSettled: () => invalidateAllButSession(queryClient),
  });
}

/** A recall of the round the person saw, with an optional reason (FR-021). */
export interface RecallInput {
  activityId: string;
  roundId: string;
  reason: string;
}

export function useRecall() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ activityId, roundId, reason }: RecallInput) =>
      call(
        api.POST('/api/v1/activities/{activityId}/approval/recall', {
          params: { path: { activityId } },
          body: reason ? { roundId, reason } : { roundId },
        }),
      ),
    onSettled: () => invalidateAllButSession(queryClient),
  });
}

/**
 * The monitoring figures of the approvals page (017 US6, FR-029, FR-030): overdue, escalated,
 * the average time to decision and the automatic approvals of this calendar month, the ageing
 * buckets, the waiting by level and the overdue rows. Asked again every 30 seconds, like the
 * counts; `enabled` false while the Monitoring tab is not shown.
 */
export function useApprovalMonitoring(enabled = true) {
  const { monthStart, monthEnd } = useMonthRange();
  return useQuery({
    queryKey: ['approval-monitoring', monthStart],
    queryFn: () =>
      call(
        api.GET('/api/v1/approvals/monitoring', { params: { query: { monthStart, monthEnd } } }),
      ),
    refetchInterval: 30_000,
    enabled,
  });
}

/** "Run reminders and escalations now" (017 US5, FR-028); every query but the session after it. */
export function useRunEscalations() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => call(api.POST('/api/v1/approvals/escalations/run')),
    onSettled: () => invalidateAllButSession(queryClient),
  });
}
