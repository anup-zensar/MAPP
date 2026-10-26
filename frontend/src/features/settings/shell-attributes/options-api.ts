// What the Options card and the option panels ask the API (feature 002, US2, US6). Every change
// refreshes what api.ts's useRefresh names, also when it failed.
import { keepPreviousData, useInfiniteQuery, useMutation } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { SETTINGS_KEYS } from '../api';
import { useRefresh } from './api';
import type { StatusChange } from './api';

export type ShellAttributeOption = components['schemas']['ShellAttributeOption'];
export type OptionImportRequest = components['schemas']['OptionImportRequest'];
export type OptionImportCheck = components['schemas']['OptionImportCheck'];
export type ImportRow = components['schemas']['ImportRow'];

export type OptionStatus = 'active' | 'inactive' | 'all';

/** The API's default page size. */
const OPTIONS_PAGE_SIZE = 100;

export interface OptionFilter {
  q: string;
  status: OptionStatus;
}

/** 100 at a time, the last answer on screen meanwhile: the API searches, so 5,000 stay fast (SC-007). */
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

/** The option as it is stored now, never from the cache (after a conflict, D-211). */
export function fetchOption(id: string) {
  return call(
    api.GET('/api/v1/shell-attribute-options/{optionId}', { params: { path: { optionId: id } } }),
  );
}

function fetchOptions(id: string, filter: OptionFilter, offset = 0) {
  return call(
    api.GET('/api/v1/shell-attributes/{attributeId}/options', {
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

export interface NewOption {
  attributeId: string;
  label: string;
  parentOptionIds?: string[];
}

export function useCreateOption() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ attributeId, label, parentOptionIds }: NewOption) =>
      call(
        api.POST('/api/v1/shell-attributes/{attributeId}/options', {
          params: { path: { attributeId } },
          body: { label, parentOptionIds },
        }),
      ),
    onSettled: () => refresh(),
  });
}

/** Only what changed (D-203), with the version the person started from. */
export interface OptionChange {
  id: string;
  label?: string;
  parentOptionIds?: string[];
  version: number;
}

export function useUpdateOption() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ id, label, parentOptionIds, version }: OptionChange) =>
      call(
        api.PATCH('/api/v1/shell-attribute-options/{optionId}', {
          params: { path: { optionId: id } },
          body: { label, parentOptionIds, version },
        }),
      ),
    onSettled: () => refresh(),
  });
}

export function useSetOptionStatus() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ id, active, version }: StatusChange) =>
      call(
        api.POST('/api/v1/shell-attribute-options/{optionId}/status', {
          params: { path: { optionId: id } },
          body: { active, version },
        }),
      ),
    onSettled: () => refresh(),
  });
}

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
        api.POST('/api/v1/shell-attribute-options/{optionId}/move', {
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
        api.DELETE('/api/v1/shell-attribute-options/{optionId}', {
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
    api.POST('/api/v1/shell-attributes/{attributeId}/options/import-check', {
      params: { path: { attributeId } },
      body,
    }),
  );
}

/** Adds the new valid options after the check (FR-031); never removes, renames or reactivates. */
export function useImportOptions() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ attributeId, file }: { attributeId: string; file: OptionImportRequest }) =>
      call(
        api.POST('/api/v1/shell-attributes/{attributeId}/options/import', {
          params: { path: { attributeId } },
          body: file,
        }),
      ),
    onSettled: () => refresh(),
  });
}
