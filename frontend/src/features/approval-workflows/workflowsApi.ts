/**
 * What Settings › Approval workflows asks the API (008 plan, "Server state"): the list, one
 * workflow's details, and every change. A workflow change shows on activities (their workflow
 * field, "(inactive)") and in counts of other features, so after every change — also a refused
 * one — every query but the session is asked again.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';
import { invalidateAllButSession } from '@/features/approvals/approvalsApi';

export type ApprovalWorkflow = components['schemas']['ApprovalWorkflow'];
export type ApprovalWorkflowDetails = components['schemas']['ApprovalWorkflowDetails'];
export type WorkflowLevelInput = components['schemas']['ApprovalWorkflowLevelInput'];
export type WorkflowCreate = components['schemas']['ApprovalWorkflowCreate'];
export type WorkflowUpdate = components['schemas']['ApprovalWorkflowUpdate'];
export type ApprovalWorkflowLevel = components['schemas']['ApprovalWorkflowLevel'];
export type ApprovalCondition = components['schemas']['ApprovalCondition'];
export type ApprovalConditionInput = components['schemas']['ApprovalConditionInput'];
export type ApprovalRuleOptions = components['schemas']['ApprovalRuleOptions'];

export const WORKFLOWS_KEY = ['approval-workflows'];

/** Every approval workflow, active and inactive, sorted by name. */
export function useWorkflows() {
  return useQuery({
    queryKey: WORKFLOWS_KEY,
    queryFn: () => call(api.GET('/api/v1/approval-workflows')),
  });
}

/** One workflow with the activities that use it. */
export function useWorkflowDetails(workflowId: string) {
  return useQuery({
    queryKey: [...WORKFLOWS_KEY, workflowId],
    queryFn: () => fetchDetails(workflowId),
  });
}

/**
 * What a level's rules may name (017 FR-001, FR-005): the activity types, the business units and
 * the roles that approve. They change rarely, so they are kept for five minutes; `enabled` false
 * waits until a level needs them.
 */
export function useRuleOptions(enabled = true) {
  return useQuery({
    queryKey: ['approval-rule-options'],
    queryFn: () => call(api.GET('/api/v1/approval-rule-options')),
    staleTime: 5 * 60_000,
    enabled,
  });
}

/** The workflow as it is stored now (after a conflict, the form takes its version). */
export async function fetchLatestWorkflow(
  queryClient: QueryClient,
  workflowId: string,
): Promise<ApprovalWorkflow> {
  const details = await queryClient.fetchQuery({
    queryKey: [...WORKFLOWS_KEY, workflowId],
    queryFn: () => fetchDetails(workflowId),
    staleTime: 0,
  });
  return details.workflow;
}

export function useCreateWorkflow() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: WorkflowCreate) => call(api.POST('/api/v1/approval-workflows', { body })),
    onSettled: refresh,
  });
}

/** Only the changed fields, with the version the person started editing from (D-203). */
export function useUpdateWorkflow() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ id, changes }: { id: string; changes: WorkflowUpdate }) =>
      call(
        api.PATCH('/api/v1/approval-workflows/{workflowId}', {
          params: { path: { workflowId: id } },
          body: changes,
        }),
      ),
    onSettled: refresh,
  });
}

/** Set inactive or active again, with the version the person saw (FR-007). */
export function useSetWorkflowActive() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ id, active, version }: { id: string; active: boolean; version: number }) => {
      const request = { params: { path: { workflowId: id } }, body: { version } };
      return call(
        active
          ? api.POST('/api/v1/approval-workflows/{workflowId}/activate', request)
          : api.POST('/api/v1/approval-workflows/{workflowId}/deactivate', request),
      );
    },
    onSettled: refresh,
  });
}

/** Moves an unused workflow to the recycle bin; answers the entry for Undo (FR-008). */
export function useDeleteWorkflow() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (id: string) =>
      call(
        api.DELETE('/api/v1/approval-workflows/{workflowId}', {
          params: { path: { workflowId: id } },
        }),
      ),
    onSettled: refresh,
  });
}

function fetchDetails(workflowId: string) {
  return call(
    api.GET('/api/v1/approval-workflows/{workflowId}', { params: { path: { workflowId } } }),
  );
}

function useRefresh() {
  const queryClient = useQueryClient();
  return () => invalidateAllButSession(queryClient);
}
