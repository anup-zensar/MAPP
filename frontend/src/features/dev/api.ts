import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient, QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';

export type SampleFolder = components['schemas']['SampleFolder'];
export type SampleNote = components['schemas']['SampleNote'];
export type RecycleBinEntry = components['schemas']['RecycleBinEntry'];
export type RecycleBinListItem = components['schemas']['RecycleBinListItem'];
export type ActivityLogEntry = components['schemas']['ActivityLogEntry'];

const LOG_ENTRIES_SHOWN = 20;

const FOLDERS_KEY: QueryKey = ['sample-folders'];
const RECYCLE_BIN_KEY: QueryKey = ['recycle-bin'];
const ACTIVITY_LOG_KEY: QueryKey = ['activity-log', { limit: LOG_ENTRIES_SHOWN }];

/** The page's three lists; a change can show in each of them. Undo refreshes them too. */
export const SAMPLE_PAGE_LISTS: QueryKey[] = [FOLDERS_KEY, RECYCLE_BIN_KEY, ACTIVITY_LOG_KEY];

export function useSampleFolders() {
  return useQuery({ queryKey: FOLDERS_KEY, queryFn: fetchFolders });
}

export function useRecycleBin() {
  return useQuery({
    queryKey: RECYCLE_BIN_KEY,
    queryFn: () => call(api.GET('/api/v1/recycle-bin')),
  });
}

export function useRecentActivity() {
  return useQuery({
    queryKey: ACTIVITY_LOG_KEY,
    queryFn: () =>
      call(api.GET('/api/v1/activity-log', { params: { query: { limit: LOG_ENTRIES_SHOWN } } })),
  });
}

export async function fetchLatestFolder(
  queryClient: QueryClient,
  id: string,
): Promise<SampleFolder | undefined> {
  // staleTime 0: always ask the API, never answer from the cache.
  const folders = await queryClient.fetchQuery({
    queryKey: FOLDERS_KEY,
    queryFn: fetchFolders,
    staleTime: 0,
  });
  return folders.items.find((folder) => folder.id === id);
}

export function useCreateFolder() {
  const refreshLists = useRefreshLists();
  return useMutation({
    mutationFn: (name: string) => call(api.POST('/api/v1/sample-folders', { body: { name } })),
    onSettled: refreshLists,
  });
}

export interface FolderRename {
  id: string;
  name: string;
  version: number;
}

/** The version comes from the edit form, never the cache, so a stale save is refused as a conflict. */
export function useRenameFolder() {
  const refreshLists = useRefreshLists();
  return useMutation({
    mutationFn: ({ id, name, version }: FolderRename) =>
      call(
        api.PATCH('/api/v1/sample-folders/{folderId}', {
          params: { path: { folderId: id } },
          body: { name, version },
        }),
      ),
    onSettled: refreshLists,
  });
}

export function useDeleteFolder() {
  const refreshLists = useRefreshLists();
  return useMutation({
    mutationFn: (id: string) =>
      call(api.DELETE('/api/v1/sample-folders/{folderId}', { params: { path: { folderId: id } } })),
    onSettled: refreshLists,
  });
}

export interface NewNote {
  folderId: string;
  title: string;
}

export function useAddNote() {
  const refreshLists = useRefreshLists();
  return useMutation({
    mutationFn: ({ folderId, title }: NewNote) =>
      call(
        api.POST('/api/v1/sample-folders/{folderId}/notes', {
          params: { path: { folderId } },
          body: { title },
        }),
      ),
    onSettled: refreshLists,
  });
}

export function useDeleteNote() {
  const refreshLists = useRefreshLists();
  return useMutation({
    mutationFn: (id: string) =>
      call(api.DELETE('/api/v1/sample-notes/{noteId}', { params: { path: { noteId: id } } })),
    onSettled: refreshLists,
  });
}

export function useRestoreFromBin() {
  const refreshLists = useRefreshLists();
  return useMutation({
    mutationFn: (entryId: string) =>
      call(api.POST('/api/v1/recycle-bin/{entryId}/restore', { params: { path: { entryId } } })),
    onSettled: refreshLists,
  });
}

function fetchFolders() {
  return call(api.GET('/api/v1/sample-folders'));
}

/** A change waits for this, so the page shows the new state when the change's message appears. */
function useRefreshLists() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(SAMPLE_PAGE_LISTS.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
