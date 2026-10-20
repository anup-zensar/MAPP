/**
 * What Field help asks the API (feature 016, US7, FR-045 – FR-048): every help entry once per
 * page load, and the edit or reset of one text. Every change refreshes the entries — and so every
 * ⓘ tip that reads them through useFieldHelp() — and the activity log.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';

export type FieldHelpEntry = components['schemas']['FieldHelpEntry'];
export type FieldHelpLevel = components['schemas']['FieldHelpLevel'];
export type FieldHelpPage = components['schemas']['FieldHelpPage'];
export type FieldHelpUpdate = components['schemas']['FieldHelpUpdate'];

/** The query key of the help entries; every tip and the Settings page read it. */
export const FIELD_HELP_KEY = ['field-help'] as const;

/** The longest help text (FR-046). */
export const HELP_MAX_LENGTH = 300;

/**
 * Every help entry with its text (GET /api/field-help). Asked again whenever a page that shows
 * tips opens (staleTime 0), so an edited text shows on the next form anyone opens (SC-008).
 */
export function useFieldHelpEntries() {
  return useQuery({
    queryKey: FIELD_HELP_KEY,
    queryFn: ({ signal }) => call(api.GET('/api/field-help', { signal })),
    staleTime: 0,
  });
}

/** One entry as it is stored now, for its version after a conflict. */
export async function fetchFieldHelp(key: string): Promise<FieldHelpEntry | undefined> {
  const page = await call(api.GET('/api/field-help'));
  return page.items.find((entry) => entry.key === key);
}

/** Saves a help text, or resets it to the default with `text: null` (FR-046). */
export function useUpdateFieldHelp() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ key, body }: { key: string; body: FieldHelpUpdate }) =>
      call(api.PATCH('/api/field-help/{helpKey}', { params: { path: { helpKey: key } }, body })),
    onSettled: () =>
      Promise.all(
        [FIELD_HELP_KEY, ['activity-log']].map((queryKey) =>
          queryClient.invalidateQueries({ queryKey }),
        ),
      ),
  });
}
