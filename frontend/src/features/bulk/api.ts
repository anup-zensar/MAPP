import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { BulkLevel, BulkRequest } from './bulk-types';

export function useBulkFields(level: BulkLevel, enabled: boolean) {
  return useQuery({
    queryKey: ['bulk', 'fields', level],
    queryFn: () => call(api.GET('/api/v1/bulk-actions/fields', { params: { query: { level } } })),
    enabled,
  });
}

export function useBulkTargets(action: 'copy' | 'move', recordIds: string[], search: string) {
  return useQuery({
    queryKey: ['bulk', 'targets', action, recordIds, search],
    queryFn: () =>
      call(api.POST('/api/v1/bulk-actions/targets', { body: { action, recordIds, search } })),
  });
}

/** What the request would do, asked afresh each time; null asks nothing yet. */
export function useBulkPreview(request: BulkRequest | null) {
  return useQuery({
    queryKey: ['bulk', 'preview', request],
    queryFn: () => call(api.POST('/api/v1/bulk-actions/preview', { body: bodyOf(request) })),
    enabled: request !== null,
    staleTime: 0,
    gcTime: 0,
  });
}

export function useRunBulk() {
  const refresh = useRefreshAfterChange();
  return useMutation({
    mutationFn: (request: BulkRequest) =>
      call(api.POST('/api/v1/bulk-actions', { body: bodyOf(request) })),
    onSuccess: refresh,
  });
}

export function useUndoBulk() {
  const refresh = useRefreshAfterChange();
  return useMutation({
    mutationFn: (bulkActionId: string) =>
      call(
        api.POST('/api/v1/bulk-actions/{bulkActionId}/undo', {
          params: { path: { bulkActionId } },
        }),
      ),
    onSuccess: refresh,
  });
}

function bodyOf(request: BulkRequest | null) {
  return { restartApproval: false, action: 'delete' as const, recordIds: [], ...request };
}

function useRefreshAfterChange() {
  const queryClient = useQueryClient();
  return () =>
    queryClient.invalidateQueries({
      predicate: ({ queryKey }) => queryKey[0] !== 'session' && queryKey[0] !== 'bulk',
    });
}
