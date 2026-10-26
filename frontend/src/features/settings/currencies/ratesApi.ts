/**
 * What the exchange-rate screens ask the API (feature 016, US1, FR-001 – FR-007): a currency's
 * rate history, adding, editing and deleting a dated rate with the impact check first, and the
 * simulated GSAP rate import (review, then import) with its recent imports and sample file.
 *
 * Every rate change refreshes everything under ['exchange-rates'] — the rate histories, the recent
 * imports and the current rates (`GET /api/v1/exchange-rates/current`, which the tips beside converted
 * amounts read under RATE_KEYS.current) — and the currencies list, whose current rates follow.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { downloadFromApi } from '@/lib/download';
import { CURRENCY_LISTS } from './currenciesApi';

export type ExchangeRateRecord = components['schemas']['ExchangeRateRecord'];
export type ExchangeRatePage = components['schemas']['ExchangeRatePage'];
export type ExchangeRateCreate = components['schemas']['ExchangeRateCreate'];
export type ExchangeRateUpdate = components['schemas']['ExchangeRateUpdate'];
export type RateUsed = components['schemas']['RateUsed'];
export type RateSource = components['schemas']['RateSource'];
export type RateImport = components['schemas']['RateImport'];
export type RateImportCheck = components['schemas']['RateImportCheck'];
export type RateImportRow = components['schemas']['RateImportRow'];
export type RateImportOutcome = components['schemas']['RateImportOutcome'];
export type RateImportCounts = components['schemas']['RateImportCounts'];
export type RateImportUpload = components['schemas']['RateImportUpload'];
export type ChangeImpact = components['schemas']['ChangeImpact'];

/** The query keys of the exchange rates; ['exchange-rates'] covers them all. */
export const RATE_KEYS = {
  all: ['exchange-rates'] as const,
  history: (code: string) => ['exchange-rates', 'history', code] as const,
  imports: ['exchange-rates', 'imports'] as const,
  /** The current rates of every currency (016 plan: useCurrentRates, read by the rate tips). */
  current: ['exchange-rates', 'current'] as const,
};

/** Recent imports shows the last 10 (FR-007). */
export const RECENT_IMPORTS = 10;

/** What a rate change or an import shows in, with the activity log and the recycle bin. */
export const RATE_LISTS: QueryKey[] = [RATE_KEYS.all, ...CURRENCY_LISTS];

/** A currency's dated rates, newest first, with its current rate (FR-003). */
export function useRateHistory(code: string) {
  return useQuery({
    queryKey: RATE_KEYS.history(code),
    queryFn: ({ signal }) =>
      call(api.GET('/api/v1/currencies/{code}/rates', { params: { path: { code } }, signal })),
  });
}

/** The last imports with their counts (FR-007). */
export function useRateImports() {
  return useQuery({
    queryKey: RATE_KEYS.imports,
    queryFn: ({ signal }) =>
      call(
        api.GET('/api/v1/exchange-rates/imports', {
          params: { query: { limit: RECENT_IMPORTS } },
          signal,
        }),
      ),
  });
}

/** What adding a rate would recalculate (FR-005). Saves nothing. */
export function checkAddRateImpact(code: string, body: ExchangeRateCreate): Promise<ChangeImpact> {
  return call(
    api.POST('/api/v1/currencies/{code}/rates/impact', { params: { path: { code } }, body }),
  );
}

/** What editing a rate would recalculate (FR-005). Saves nothing. */
export function checkUpdateRateImpact(
  rateId: string,
  body: ExchangeRateUpdate,
): Promise<ChangeImpact> {
  return call(
    api.POST('/api/v1/exchange-rates/{rateId}/impact', { params: { path: { rateId } }, body }),
  );
}

/** One rate as it is stored now, for its version after a conflict. */
export async function fetchRate(code: string, rateId: string) {
  const page = await call(
    api.GET('/api/v1/currencies/{code}/rates', { params: { path: { code } } }),
  );
  return page.items.find((rate) => rate.id === rateId);
}

export function useAddRate() {
  const refresh = useRefreshRateLists();
  return useMutation({
    mutationFn: ({ code, body }: { code: string; body: ExchangeRateCreate }) =>
      call(api.POST('/api/v1/currencies/{code}/rates', { params: { path: { code } }, body })),
    onSettled: refresh,
  });
}

/** Changes a rate's value (its date never changes); the rate becomes "Entered by hand". */
export function useUpdateRate() {
  const refresh = useRefreshRateLists();
  return useMutation({
    mutationFn: ({ rateId, body }: { rateId: string; body: ExchangeRateUpdate }) =>
      call(api.PATCH('/api/v1/exchange-rates/{rateId}', { params: { path: { rateId } }, body })),
    onSettled: refresh,
  });
}

/** Moves a rate to the recycle bin; the last rate dated today or earlier is refused. */
export function useDeleteRate() {
  const refresh = useRefreshRateLists();
  return useMutation({
    mutationFn: (rateId: string) =>
      call(api.DELETE('/api/v1/exchange-rates/{rateId}', { params: { path: { rateId } } })),
    onSettled: refresh,
  });
}

/** Reviews a rate file: every row's outcome, nothing saved (FR-006). */
export function checkRateImport(body: RateImportUpload): Promise<RateImportCheck> {
  return call(api.POST('/api/v1/exchange-rates/import-check', { body }));
}

/** Imports the rows to add and replace of a rate file in one step (FR-006, FR-007). */
export function useImportRates() {
  const refresh = useRefreshRateLists();
  return useMutation({
    mutationFn: (body: RateImportUpload) =>
      call(api.POST('/api/v1/exchange-rates/import', { body })),
    onSettled: refresh,
  });
}

/** Saves the sample file of today's rates; answers its name. */
export function downloadRateSample(): Promise<string> {
  return downloadFromApi(
    api.GET('/api/v1/exchange-rates/import-sample', { parseAs: 'blob' }),
    'gsap_rates_sample.csv',
  );
}

function useRefreshRateLists() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(RATE_LISTS.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
