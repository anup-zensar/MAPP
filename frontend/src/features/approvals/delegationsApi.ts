/**
 * Delegation of approval authority (017 US4, FR-022 – FR-024): who approves for whom, from when to
 * when. A delegation changes who waits for what (Waiting for me, the counts, the chain), so after
 * every change — also a refused one — every query but the session is asked again (008's rule).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';
import { invalidateAllButSession } from './approvalsApi';

export type ApprovalDelegation = components['schemas']['ApprovalDelegation'];
export type ApprovalDelegationPage = components['schemas']['ApprovalDelegationPage'];
export type DelegationCreate = components['schemas']['ApprovalDelegationCreate'];
export type DelegationUpdate = components['schemas']['ApprovalDelegationUpdate'];

/** Mine: the signed-in person's delegations, given and received; all: everyone's (workflows.manage). */
export type DelegationScope = 'mine' | 'all';

export function delegationsKey(scope: DelegationScope) {
  return ['approval-delegations', scope];
}

/** The delegations, current and planned first, then ended and cancelled (the API's order). */
export function useDelegations(scope: DelegationScope = 'mine', enabled = true) {
  return useQuery({
    queryKey: delegationsKey(scope),
    queryFn: () => fetchDelegations(scope),
    enabled,
  });
}

/** The delegation as it is stored now (after a conflict, the form takes its version). */
export async function fetchLatestDelegation(
  queryClient: QueryClient,
  scope: DelegationScope,
  id: string,
): Promise<ApprovalDelegation | undefined> {
  const page = await queryClient.fetchQuery({
    queryKey: delegationsKey(scope),
    queryFn: () => fetchDelegations(scope),
    staleTime: 0,
  });
  return page.items.find((delegation) => delegation.id === id);
}

function fetchDelegations(scope: DelegationScope) {
  return call(api.GET('/api/v1/approval-delegations', { params: { query: { scope } } }));
}

export function useCreateDelegation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: DelegationCreate) =>
      call(api.POST('/api/v1/approval-delegations', { body })),
    onSettled: () => invalidateAllButSession(queryClient),
  });
}

/** Only the changed fields, with the version the person started from. */
export function useUpdateDelegation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, changes }: { id: string; changes: DelegationUpdate }) =>
      call(
        api.PATCH('/api/v1/approval-delegations/{delegationId}', {
          params: { path: { delegationId: id } },
          body: changes,
        }),
      ),
    onSettled: () => invalidateAllButSession(queryClient),
  });
}

export function useCancelDelegation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, version }: { id: string; version: number }) =>
      call(
        api.POST('/api/v1/approval-delegations/{delegationId}/cancel', {
          params: { path: { delegationId: id } },
          body: { version },
        }),
      ),
    onSettled: () => invalidateAllButSession(queryClient),
  });
}
