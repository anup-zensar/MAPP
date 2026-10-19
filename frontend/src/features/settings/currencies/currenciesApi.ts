/**
 * What the Currencies page asks the API (feature 002, US4). The list comes from useCurrencies()
 * (api/settingsQueries.ts); every change refreshes it and the activity log, recycle bin and
 * Settings counts.
 */
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';
import { SETTINGS_KEYS } from '@/api/settingsQueries';

export type Currency = components['schemas']['Currency'];
export type CurrencyCreate = components['schemas']['CurrencyCreate'];
export type CurrencyUpdate = components['schemas']['CurrencyUpdate'];
export type ChangeImpact = components['schemas']['ChangeImpact'];

/** The lists a change to a currency shows in. */
export const CURRENCY_LISTS: QueryKey[] = [
  SETTINGS_KEYS.currencies,
  SETTINGS_KEYS.overview,
  ['activity-log'],
  ['recycle-bin'],
];

/** One currency as it is stored now; after a conflict the form takes its version from it. */
export function fetchCurrency(code: string): Promise<Currency> {
  return call(api.GET('/api/currencies/{code}', { params: { path: { code } } }));
}

/** The line items and activities a rate change would recalculate (FR-030). Saves nothing. */
export function checkCurrencyImpact(code: string, body: CurrencyUpdate): Promise<ChangeImpact> {
  return call(api.POST('/api/currencies/{code}/impact', { params: { path: { code } }, body }));
}

/** POST …/status: sets a currency inactive or reactivates it (FR-004). */
export function setCurrencyStatus(code: string, body: { active: boolean; version: number }) {
  return call(api.POST('/api/currencies/{code}/status', { params: { path: { code } }, body }));
}

export function useCreateCurrency() {
  const refresh = useRefreshCurrencyLists();
  return useMutation({
    mutationFn: (body: CurrencyCreate) => call(api.POST('/api/currencies', { body })),
    onSettled: refresh,
  });
}

/** Changes a currency: only the fields sent, with the version the form started from (D-203). */
export function useUpdateCurrency() {
  const refresh = useRefreshCurrencyLists();
  return useMutation({
    mutationFn: ({ code, body }: { code: string; body: CurrencyUpdate }) =>
      call(api.PATCH('/api/currencies/{code}', { params: { path: { code } }, body })),
    onSettled: refresh,
  });
}

/** Deletes an unused currency (to the recycle bin). */
export function useDeleteCurrency() {
  const refresh = useRefreshCurrencyLists();
  return useMutation({
    mutationFn: (code: string) =>
      call(api.DELETE('/api/currencies/{code}', { params: { path: { code } } })),
    onSettled: refresh,
  });
}

function useRefreshCurrencyLists() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(CURRENCY_LISTS.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
