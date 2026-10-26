/**
 * Which exchange rates an amount was converted with (016 FR-009): "1 USD = 0.79 GBP · rate of
 * 19 Oct 2026 · GSAP import". The web app never converts on its own for display (D-061); it only
 * says which rate the API used. Until feature 016's shared `useCurrentRates()` and `RateTip` exist,
 * the activity and line-item pages read the current rates here.
 */
import { useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { formatRate } from '@/format/decimal';

export type RateUsed = components['schemas']['RateUsed'];
type RateSource = components['schemas']['RateSource'];

/** The current rate of every currency, with its date and source (GET /api/v1/exchange-rates/current). */
export function useCurrentRates(enabled = true) {
  return useQuery({
    queryKey: ['exchange-rates', 'current'],
    queryFn: () => call(api.GET('/api/v1/exchange-rates/current')),
    // Rates change at most once a day; a rate save or import asks again (016 plan).
    staleTime: 5 * 60_000,
    enabled,
  });
}

/** Where a rate came from, in words. */
export function rateSourceText(source: RateSource | null | undefined): string {
  return source === 'gsapImport' ? 'GSAP import' : 'Settings';
}

/** One currency's rate as a sentence part (FR-009). */
export interface RateFacts {
  code: string;
  /** Units of the currency per 1 USD. */
  ratePerUsd: string;
  /** The rate's date ("YYYY-MM-DD") when known. */
  rateDate?: string | null;
  source?: RateSource | null;
}

/**
 * "1 USD = 0.79 GBP · rate of 19 Oct 2026 · GSAP import"; without a known date "1 USD = 0.79 GBP ·
 * today's rate in Settings". USD itself needs no sentence (null).
 */
export function rateText(rate: RateFacts, date: (day: string) => string): string | null {
  if (rate.code === 'USD') {
    return null;
  }
  const head = `1 USD = ${formatRate(rate.ratePerUsd)} ${rate.code}`;
  if (rate.rateDate) {
    return `${head} · rate of ${date(rate.rateDate)} · ${rateSourceText(rate.source)}`;
  }
  return `${head} · today's rate in Settings`;
}

/** The current rate of a currency from GET /api/v1/exchange-rates/current, as RateFacts. */
export function factsOf(rate: RateUsed): RateFacts {
  return {
    code: rate.currency,
    ratePerUsd: rate.rate,
    rateDate: rate.rateDate,
    source: rate.source,
  };
}

/** Said when the rates themselves could not be read. */
export const RATES_UNKNOWN = "Converted at today's rates in Settings.";
