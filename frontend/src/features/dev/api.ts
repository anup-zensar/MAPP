/**
 * What the sample records page asks the API (research R23): its three lists — the sample folders,
 * the recycle bin and the newest activity-log entries — and every change it can make.
 *
 * Every change refreshes the three lists when it is done, also when it failed: a delete shows in
 * all three, and a refused save may mean that someone else changed the records meanwhile.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient, QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';

export type SampleFolder = components['schemas']['SampleFolder'];
export type SampleNote = components['schemas']['SampleNote'];
export type RecycleBinEntry = components['schemas']['RecycleBinEntry'];
export type RecycleBinListItem = components['schemas']['RecycleBinListItem'];
export type ActivityLogEntry = components['schemas']['ActivityLogEntry'];

/** How many activity-log entries the page shows (plan "Developer pages"). */
const LOG_ENTRIES_SHOWN = 20;

const FOLDERS_KEY: QueryKey = ['sample-folders'];
const RECYCLE_BIN_KEY: QueryKey = ['recycle-bin'];
const ACTIVITY_LOG_KEY: QueryKey = ['activity-log', { limit: LOG_ENTRIES_SHOWN }];

/** The page's three lists; a change can show in each of them. Undo refreshes them too. */
export const SAMPLE_PAGE_LISTS: QueryKey[] = [FOLDERS_KEY, RECYCLE_BIN_KEY, ACTIVITY_LOG_KEY];

/** Every sample folder with its notes, sorted by name. */
export function useSampleFolders() {
  return useQuery({ queryKey: FOLDERS_KEY, queryFn: fetchFolders });
}

/** Everything in the recycle bin that can still be restored, newest first. */
export function useRecycleBin() {
  return useQuery({
    queryKey: RECYCLE_BIN_KEY,
    queryFn: () => call(api.GET('/api/v1/recycle-bin')),
  });
}

/** The newest activity-log entries, newest first. */
export function useRecentActivity() {
  return useQuery({
    queryKey: ACTIVITY_LOG_KEY,
    queryFn: () =>
      call(api.GET('/api/v1/activity-log', { params: { query: { limit: LOG_ENTRIES_SHOWN } } })),
  });
}

/**
 * Asks for the folders again and returns the folder `id` as it is stored now, or undefined when
 * it no longer exists. After a conflict, the edit form takes the latest version from it.
 */
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

/** Creates a sample folder with the given name. */
export function useCreateFolder() {
  const refreshLists = useRefreshLists();
  return useMutation({
    mutationFn: (name: string) => call(api.POST('/api/v1/sample-folders', { body: { name } })),
    onSettled: refreshLists,
  });
}

/** A new name for a folder, and the version of the folder the person started editing from. */
export interface FolderRename {
  id: string;
  name: string;
  version: number;
}

/**
 * Renames a folder. The version comes from the edit form and never from the cache (the edit-form
 * rule, plan "Shared hooks"): when someone else changed the folder after the form opened, the API
 * refuses the save with a conflict instead of overwriting their change (FR-029).
 */
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

/** Moves a folder, with its notes, to the recycle bin; answers the recycle-bin entry for Undo. */
export function useDeleteFolder() {
  const refreshLists = useRefreshLists();
  return useMutation({
    mutationFn: (id: string) =>
      call(api.DELETE('/api/v1/sample-folders/{folderId}', { params: { path: { folderId: id } } })),
    onSettled: refreshLists,
  });
}

/** A new note: the folder it goes in and its title. */
export interface NewNote {
  folderId: string;
  title: string;
}

/** Adds a note to a folder. */
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

/** Moves a note to the recycle bin; answers the recycle-bin entry for Undo. */
export function useDeleteNote() {
  const refreshLists = useRefreshLists();
  return useMutation({
    mutationFn: (id: string) =>
      call(api.DELETE('/api/v1/sample-notes/{noteId}', { params: { path: { noteId: id } } })),
    onSettled: refreshLists,
  });
}

/** Brings back a recycle-bin entry: the record and everything that was under it (FR-025). */
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

/**
 * Asks for the page's three lists again. A change waits for this before it is done, so the page
 * already shows the new state when the change's message appears.
 */
function useRefreshLists() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(SAMPLE_PAGE_LISTS.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
