/**
 * What the recycle bin page asks the API (feature 011, plan "Query keys"): the bin, a restore's
 * preview, and the three changes — restore, delete permanently and empty.
 *
 * A change can touch any page (a restore brings back plans, totals, Home and approvals), so after
 * each one every query is asked again except the session and the restore previews (whose entry
 * may now be gone).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';

const BIN_KEY = ['recycle-bin'];

/** Every unresolved recycle-bin entry, newest first, with its restore state and contents. */
export function useRecycleBin() {
  return useQuery({
    queryKey: BIN_KEY,
    queryFn: () => call(api.GET('/api/v1/recycle-bin')),
  });
}

/** What restoring the entry would do, asked afresh each time the dialog opens (FR-018). */
export function useRestorePreview(entryId: string, enabled = true) {
  return useQuery({
    queryKey: ['recycle-bin', 'preview', entryId],
    queryFn: () =>
      call(
        api.GET('/api/v1/recycle-bin/{entryId}/restore-preview', { params: { path: { entryId } } }),
      ),
    enabled,
    staleTime: 0,
    gcTime: 0,
  });
}

/** Restores an entry; with `withParents`, the entries holding its parents first (FR-023). */
export function useRestore() {
  const refreshAll = useRefreshAll();
  return useMutation({
    mutationFn: ({ entryId, withParents }: { entryId: string; withParents: boolean }) =>
      call(
        api.POST('/api/v1/recycle-bin/{entryId}/restore', {
          params: { path: { entryId }, query: { withParents } },
        }),
      ),
    onSettled: refreshAll,
  });
}

/** Deletes an entry permanently, with everything inside it (FR-028, FR-030). */
export function usePurge() {
  const refreshAll = useRefreshAll();
  return useMutation({
    mutationFn: (entryId: string) =>
      call(api.DELETE('/api/v1/recycle-bin/{entryId}', { params: { path: { entryId } } })),
    onSettled: refreshAll,
  });
}

/** Deletes permanently the entries the person saw in the confirmation (FR-029). */
export function useEmptyBin() {
  const refreshAll = useRefreshAll();
  return useMutation({
    mutationFn: (entryIds: string[]) =>
      call(api.POST('/api/v1/recycle-bin/empty', { body: { entryIds } })),
    onSettled: refreshAll,
  });
}

/** Asks every query again, except the session and the restore previews. */
function useRefreshAll() {
  const queryClient = useQueryClient();
  return () =>
    queryClient.invalidateQueries({
      predicate: ({ queryKey }) =>
        queryKey[0] !== 'session' && !(queryKey[0] === 'recycle-bin' && queryKey[1] === 'preview'),
    });
}
