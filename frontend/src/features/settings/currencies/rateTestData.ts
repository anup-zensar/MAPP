/**
 * Exchange rates, rate imports and a reviewed GSAP rate file for the tests of this folder, shaped
 * as the API answers them (backend/api/openapi.yaml, feature 016). Only the tests use them.
 */
import type { components } from '@/api/generated/schema';
import type {
  ExchangeRatePage,
  ExchangeRateRecord,
  RateImport,
  RateImportCheck,
  RateImportRow,
} from './ratesApi';

type ActorRef = components['schemas']['ActorRef'];

export const ALEX_REF: ActorRef = { id: 'U-1001', name: 'Alex Morgan' };
export const DANIEL_REF: ActorRef = { id: 'U-1003', name: 'Daniel Okoro' };

function rate(
  overrides: Partial<ExchangeRateRecord> & { id: string; date: string },
): ExchangeRateRecord {
  return {
    currency: 'EUR',
    rate: '0.92',
    inverseRate: '1.086957',
    source: 'gsapImport',
    importId: 'RI-1001',
    importFileName: 'gsap_rates_2026-10-19.csv',
    current: false,
    future: false,
    version: 1,
    createdAt: '2026-10-19T06:00:00Z',
    createdBy: ALEX_REF,
    updatedAt: '2026-10-19T06:00:00Z',
    updatedBy: ALEX_REF,
    ...overrides,
  };
}

/** 0.95 from 1 Jan 2027, entered by hand by Daniel. */
export const EUR_FUTURE = rate({
  id: 'XR-1103',
  date: '2027-01-01',
  rate: '0.95',
  inverseRate: '1.052632',
  source: 'manual',
  importId: null,
  importFileName: null,
  future: true,
  createdBy: DANIEL_REF,
  updatedBy: DANIEL_REF,
  createdAt: '2026-10-20T09:15:00Z',
  updatedAt: '2026-10-20T09:15:00Z',
});

/** The rate in use: 0.92 of 19 Oct 2026, from the GSAP import. */
export const EUR_CURRENT = rate({ id: 'XR-1102', date: '2026-10-19', current: true });

/** An older imported rate of 18 Oct 2026. */
export const EUR_OLDER = rate({
  id: 'XR-1101',
  date: '2026-10-18',
  rate: '0.9214',
  inverseRate: '1.085305',
  importFileName: 'gsap_rates_2026-10-18.csv',
  createdAt: '2026-10-18T06:00:00Z',
  updatedAt: '2026-10-18T06:00:00Z',
});

const CURRENT_EUR = {
  currency: 'EUR',
  rate: '0.92',
  inverseRate: '1.086957',
  rateDate: '2026-10-19',
  source: 'gsapImport' as const,
  importFileName: 'gsap_rates_2026-10-19.csv',
};

/** EUR's rate history, newest first. */
export const EUR_RATES: ExchangeRatePage = {
  items: [EUR_FUTURE, EUR_CURRENT, EUR_OLDER],
  total: 3,
  current: CURRENT_EUR,
};

/** EUR with only its current rate: it can't be deleted. */
export const EUR_ONLY_CURRENT: ExchangeRatePage = {
  items: [EUR_CURRENT],
  total: 1,
  current: CURRENT_EUR,
};

/** A currency without dated rates yet (an API before 016). */
export const NO_RATES: ExchangeRatePage = {
  items: [],
  total: 0,
  current: {
    currency: 'JPY',
    rate: '150',
    inverseRate: '0.006667',
    rateDate: null,
    source: null,
    importFileName: null,
  },
};

/** The import of 20 Oct 2026 by Alex. */
export const IMPORT_20_OCT: RateImport = {
  id: 'RI-1002',
  fileName: 'gsap_rates_2026-10-20.csv',
  feed: 'gsapDailyRatesFile',
  firstDate: '2026-09-21',
  lastDate: '2026-10-20',
  added: 87,
  replaced: 2,
  unchanged: 1,
  notUsed: 1,
  problems: 1,
  importedAt: '2026-10-20T08:30:00Z',
  importedBy: ALEX_REF,
};

export const RECENT_IMPORTS_PAGE = { items: [IMPORT_20_OCT], total: 1 };

export const NO_IMPORTS = { items: [], total: 0 };

function row(overrides: Partial<RateImportRow> & { row: number }): RateImportRow {
  return {
    outcome: 'add',
    currency: 'EUR',
    date: '2026-10-20',
    rate: '0.9230',
    previousRate: null,
    message: null,
    ...overrides,
  };
}

/** The review of spec US1-2, with fewer rows to add: 3 to add, 2 replace, 1 of each other. */
export const REVIEW: RateImportCheck = {
  fileName: 'gsap_rates_2026-10-20.csv',
  feed: 'gsapDailyRatesFile',
  rows: [
    row({ row: 2 }),
    row({ row: 3, currency: 'GBP', rate: '0.7912' }),
    row({ row: 4, currency: 'INR', rate: '83.61' }),
    row({ row: 5, outcome: 'replace', date: '2026-10-19', rate: '0.9230', previousRate: '0.9214' }),
    row({
      row: 6,
      outcome: 'replace',
      currency: 'GBP',
      date: '2026-10-19',
      rate: '0.7905',
      previousRate: '0.79',
    }),
    row({ row: 7, outcome: 'present', currency: 'CNY', rate: '7.1' }),
    row({
      row: 8,
      outcome: 'notUsed',
      currency: 'CHF',
      rate: '0.8812',
      message: 'CHF is not set up in Settings › Currencies.',
    }),
    row({
      row: 41,
      outcome: 'problem',
      currency: 'EUR',
      rate: null,
      message: 'Row 41: the rate must be above 0 with at most 6 decimals.',
    }),
  ],
  counts: { add: 3, replace: 2, present: 1, notUsed: 1, problem: 1 },
};

/** A file whose rates are all present already. */
export const REVIEW_NOTHING_NEW: RateImportCheck = {
  fileName: 'gsap_rates_2026-10-19.csv',
  feed: 'gsapDailyRatesFile',
  rows: [row({ row: 2, outcome: 'present', date: '2026-10-19', rate: '0.92' })],
  counts: { add: 0, replace: 0, present: 1, notUsed: 0, problem: 0 },
};

/** What the import answers for REVIEW. */
export const IMPORTED: RateImport = {
  ...IMPORT_20_OCT,
  id: 'RI-1003',
  added: 3,
  replaced: 2,
};
