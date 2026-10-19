/**
 * What the Shell attribute pages ask the API (feature 002, US2): the attributes of every level,
 * one attribute, its options page by page, and every change to attributes and options.
 *
 * Every change refreshes what it can affect when it is done, also when it failed (a refused save
 * may mean someone else changed the attribute meanwhile): the lists, the attribute, its options,
 * the forms of the levels (SC-008), the activity log, the recycle bin and the Settings counts.
 */
import {
  keepPreviousData,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';
import { SETTINGS_KEYS } from '@/api/settingsQueries';
import { LEVELS } from './attributeWords';

export type ShellAttribute = components['schemas']['ShellAttribute'];
export type ShellAttributeList = components['schemas']['ShellAttributePage'];
export type ShellAttributeOption = components['schemas']['ShellAttributeOption'];
export type ShellAttributeCreate = components['schemas']['ShellAttributeCreate'];
export type ShellAttributeUpdate = components['schemas']['ShellAttributeUpdate'];
export type ChangeImpact = components['schemas']['ChangeImpact'];
export type Usage = components['schemas']['Usage'];
export type DeleteResult = components['schemas']['DeleteResult'];
export type AttributeCheckRequest = components['schemas']['AttributeCheckRequest'];
export type AttributeCheckResult = components['schemas']['AttributeCheckResult'];
export type OptionImportRequest = components['schemas']['OptionImportRequest'];
export type OptionImportCheck = components['schemas']['OptionImportCheck'];
export type ImportRow = components['schemas']['ImportRow'];

/** The status filter of the options list. */
export type OptionStatus = 'active' | 'inactive' | 'all';

/** How many options one page of the options list holds (the API's default). */
export const OPTIONS_PAGE_SIZE = 100;

/** Every level's attributes: the Settings list, its level counts and the editor's sections. */
const ALL_ATTRIBUTES_KEY = ['shell-attributes', 'all'] as const;

/** The Settings list of attributes (every key that starts with it). */
const LISTS: QueryKey = ['shell-attributes'];
/** The pages of single attributes. */
const ATTRIBUTE_PAGES: QueryKey = ['shell-attribute'];
/** What else a change to attributes or options can make out of date. */
const ELSEWHERE: QueryKey[] = [
  ['shell-attribute-options'],
  ['shell-attribute-form'],
  ['activity-log'],
  ['recycle-bin'],
  SETTINGS_KEYS.overview,
];

/** Everything a change to attributes or options can make out of date; an Undo brings it back. */
export const CHANGED_BY_ANY_CHANGE: QueryKey[] = [LISTS, ATTRIBUTE_PAGES, ...ELSEWHERE];

/** Every Shell attribute of every level, in display order (GET /api/shell-attributes). */
export function useAllShellAttributes() {
  return useQuery({
    queryKey: ALL_ATTRIBUTES_KEY,
    queryFn: () => call(api.GET('/api/shell-attributes')),
  });
}

/** One Shell attribute with its usage and locks. */
export function useShellAttribute(id: string) {
  return useQuery({
    queryKey: SETTINGS_KEYS.shellAttribute(id),
    queryFn: () => fetchAttribute(id),
  });
}

/** The attribute as it is stored now, never from the cache (after a conflict). */
export function fetchAttribute(id: string) {
  return call(
    api.GET('/api/shell-attributes/{attributeId}', { params: { path: { attributeId: id } } }),
  );
}

/** The search and status filter of the options list. */
export interface OptionFilter {
  q: string;
  status: OptionStatus;
}

/**
 * The attribute's options that match the filter, in display order, 100 at a time (FR-018,
 * SC-007): the API searches, so 5,000 options stay fast. fetchNextPage adds the next 100. While
 * a new search is on its way, the last answer stays on screen.
 */
export function useShellAttributeOptions(id: string, filter: OptionFilter) {
  return useInfiniteQuery({
    queryKey: SETTINGS_KEYS.shellAttributeOptions(id, filter),
    queryFn: ({ pageParam }) => fetchOptions(id, filter, pageParam),
    initialPageParam: 0,
    getNextPageParam: (lastPage, pages) => {
      const loaded = pages.reduce((count, page) => count + page.items.length, 0);
      return loaded < lastPage.total && lastPage.items.length > 0 ? loaded : undefined;
    },
    placeholderData: keepPreviousData,
  });
}

/** One page of matching options from `offset` on. */
export function fetchOptions(id: string, filter: OptionFilter, offset = 0) {
  return call(
    api.GET('/api/shell-attributes/{attributeId}/options', {
      params: {
        path: { attributeId: id },
        query: {
          q: filter.q.trim() || undefined,
          status: filter.status,
          limit: OPTIONS_PAGE_SIZE,
          offset,
        },
      },
    }),
  );
}

/** Creates a Shell attribute with its first options. */
export function useCreateAttribute() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: ShellAttributeCreate) => call(api.POST('/api/shell-attributes', { body })),
    onSettled: () => refresh(),
  });
}

/** A change to an attribute: only the changed fields, and the version the person started from. */
export interface AttributeChange {
  id: string;
  body: ShellAttributeUpdate;
}

/** Saves the changed fields of an attribute (D-203). */
export function useUpdateAttribute() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ id, body }: AttributeChange) =>
      call(
        api.PATCH('/api/shell-attributes/{attributeId}', {
          params: { path: { attributeId: id } },
          body,
        }),
      ),
    onSettled: () => refresh(),
  });
}

/** What the change would affect, before saving (FR-021); saves nothing. */
export function checkAttributeImpact({ id, body }: AttributeChange): Promise<ChangeImpact> {
  return call(
    api.POST('/api/shell-attributes/{attributeId}/impact', {
      params: { path: { attributeId: id } },
      body,
    }),
  );
}

/** A status change: the record, the new status and the version the person started from. */
export interface StatusChange {
  id: string;
  active: boolean;
  version: number;
}

/** Sets an attribute inactive or reactivates it (FR-004, FR-023). */
export function useSetAttributeStatus() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ id, active, version }: StatusChange) =>
      call(
        api.POST('/api/shell-attributes/{attributeId}/status', {
          params: { path: { attributeId: id } },
          body: { active, version },
        }),
      ),
    onSettled: () => refresh(),
  });
}

/** A move of an attribute, or of its whole section, one place up or down (FR-019). */
export interface AttributeMove {
  attribute: ShellAttribute;
  what: 'attribute' | 'section';
  direction: 'up' | 'down';
}

/**
 * Moves an attribute or its section. The answer is the level's new order, which replaces that
 * level in the list at once, so the moved row is in its new place when the move is done.
 */
export function useMoveAttribute() {
  const queryClient = useQueryClient();
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ attribute, what, direction }: AttributeMove) =>
      call(
        api.POST('/api/shell-attributes/{attributeId}/move', {
          params: { path: { attributeId: attribute.id } },
          body: { what, direction, version: attribute.version },
        }),
      ),
    onSuccess: (levelList, { attribute }) => {
      queryClient.setQueryData<ShellAttributeList>(ALL_ATTRIBUTES_KEY, (all) =>
        all ? withLevelReplaced(all, attribute.level, levelList) : all,
      );
    },
    // After a move the list is already up to date; everything else is asked again.
    onSettled: (_data, error) => refresh(error ? undefined : [ATTRIBUTE_PAGES, ...ELSEWHERE]),
  });
}

/** Moves an attribute and its options to the recycle bin; answers the entry for Undo. */
export function useDeleteAttribute() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (id: string) =>
      call(
        api.DELETE('/api/shell-attributes/{attributeId}', {
          params: { path: { attributeId: id } },
        }),
      ),
    // Not the deleted attribute's own page: it would ask for an attribute that is gone.
    onSettled: () => refresh([LISTS, ...ELSEWHERE]),
  });
}

/**
 * A new option: the attribute it belongs to, its label and — for a dependent attribute — the
 * parent options it is linked to (FR-022).
 */
export interface NewOption {
  attributeId: string;
  label: string;
  parentOptionIds?: string[];
}

/** Adds an option at the end of the attribute's list (FR-018). */
export function useCreateOption() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ attributeId, label, parentOptionIds }: NewOption) =>
      call(
        api.POST('/api/shell-attributes/{attributeId}/options', {
          params: { path: { attributeId } },
          body: { label, parentOptionIds },
        }),
      ),
    onSettled: () => refresh(),
  });
}

/**
 * A change to an option: a new label and/or new parent links (only what changed, D-203), and the
 * version the person started from.
 */
export interface OptionChange {
  id: string;
  label?: string;
  parentOptionIds?: string[];
  version: number;
}

/** Renames an option or changes its links to the parent's options (FR-018, FR-022). */
export function useUpdateOption() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ id, label, parentOptionIds, version }: OptionChange) =>
      call(
        api.PATCH('/api/shell-attribute-options/{optionId}', {
          params: { path: { optionId: id } },
          body: { label, parentOptionIds, version },
        }),
      ),
    onSettled: () => refresh(),
  });
}

/** Sets an option inactive or reactivates it (FR-018). */
export function useSetOptionStatus() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ id, active, version }: StatusChange) =>
      call(
        api.POST('/api/shell-attribute-options/{optionId}/status', {
          params: { path: { optionId: id } },
          body: { active, version },
        }),
      ),
    onSettled: () => refresh(),
  });
}

/** Moves an option one place up or down in its list (FR-018). */
export function useMoveOption() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({
      option,
      direction,
    }: {
      option: ShellAttributeOption;
      direction: 'up' | 'down';
    }) =>
      call(
        api.POST('/api/shell-attribute-options/{optionId}/move', {
          params: { path: { optionId: option.id } },
          body: { direction, version: option.version },
        }),
      ),
    onSettled: () => refresh(),
  });
}

/** Moves an unused option to the recycle bin; answers the entry for Undo. */
export function useDeleteOption() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (id: string) =>
      call(
        api.DELETE('/api/shell-attribute-options/{optionId}', {
          params: { path: { optionId: id } },
        }),
      ),
    onSettled: () => refresh(),
  });
}

/** An option file of an attribute: what its rows would do, before anything is added (FR-031). */
export function checkOptionImport(
  attributeId: string,
  body: OptionImportRequest,
): Promise<OptionImportCheck> {
  return call(
    api.POST('/api/shell-attributes/{attributeId}/options/import-check', {
      params: { path: { attributeId } },
      body,
    }),
  );
}

/**
 * Adds the new valid options of an option file, after its check (FR-031); never removes, renames
 * or reactivates options.
 */
export function useImportOptions() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ attributeId, file }: { attributeId: string; file: OptionImportRequest }) =>
      call(
        api.POST('/api/shell-attributes/{attributeId}/options/import', {
          params: { path: { attributeId } },
          body: file,
        }),
      ),
    onSettled: () => refresh(),
  });
}

/**
 * What saving these values on a record of `level` would say (the preview's **Try saving**,
 * FR-020): the problems, the dependent values it would clear and the fields still needed. Saves
 * nothing.
 */
export function checkAttributeValues(
  level: ShellAttribute['level'],
  body: AttributeCheckRequest,
): Promise<AttributeCheckResult> {
  return call(
    api.POST('/api/shell-attribute-forms/{level}/check', { params: { path: { level } }, body }),
  );
}

/** The attributes of every level, with `level`'s replaced by its new order. */
function withLevelReplaced(
  all: ShellAttributeList,
  level: ShellAttribute['level'],
  levelList: ShellAttributeList,
): ShellAttributeList {
  const items = LEVELS.flatMap((each) =>
    each === level ? levelList.items : all.items.filter((attribute) => attribute.level === each),
  );
  return { items, total: items.length };
}

/**
 * Asks again for what a change can affect (only the queries on screen are asked at once). A
 * change waits for this, so the page shows the new state when the change's message appears.
 */
function useRefresh() {
  const queryClient = useQueryClient();
  return (keys: QueryKey[] = CHANGED_BY_ANY_CHANGE) =>
    Promise.all(keys.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
