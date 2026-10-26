import { z } from 'zod';
import {
  ACTIVITY_STATUS_LABELS,
  CONTAINER_STATUS_LABELS,
  FINANCE_LABELS,
  isLabelled,
  MAX_YEARS,
  PLAN_STATUS_LABELS,
  PLAN_TYPE_LABELS,
  SHOW_LEVELS_LABELS,
  STRATEGIC_LABELS,
} from './all-work-text';
import type { FilterValue, Filters, SavedView, ViewState } from './all-work-types';
import { NAME_MAX, nameTaken, viewNamed } from './view-names';
import { NO_FILTERS, withYears } from './view-state';

/** The address parameters other pages may link with (the API's names, D-122). */
const ADDRESS_KEYS = [
  'year',
  'yearFrom',
  'yearTo',
  'planType',
  'planCurrency',
  'finance',
  'plan',
  'planStatus',
  'containerStatus',
  'activityStatus',
  'owner',
  'team',
  'businessUnit',
  'activityType',
  'strategic',
  'showLevels',
  'search',
] as const;

/** A plan currency in the address: three capital letters. */
const CURRENCY_CODE = /^[A-Z]{3}$/;

const yearParam = z.coerce.number().int().min(2000).max(2100).optional().catch(undefined);

const ids = z
  .array(z.string())
  .transform((values) => values.map((id): FilterValue => ({ id, label: id })));

function known<T extends string>(labels: Record<T, string>) {
  return z.array(z.string()).transform((values) => values.filter(isLabelled(labels)));
}

function firstKnown<T extends string>(labels: Record<T, string>, fallback: NoInfer<T>) {
  return known(labels).transform((values) => values.at(0) ?? fallback);
}

const currencyCodes = z
  .array(z.string())
  .transform((codes) => codes.filter((code) => CURRENCY_CODE.test(code)));

const addressSchema = z.object({
  year: yearParam,
  yearFrom: yearParam,
  yearTo: yearParam,
  plans: ids,
  planStatuses: known(PLAN_STATUS_LABELS),
  containerStatuses: known(CONTAINER_STATUS_LABELS),
  activityStatuses: known(ACTIVITY_STATUS_LABELS),
  owners: ids,
  teams: ids,
  businessUnits: ids,
  activityTypes: ids,
  strategic: firstKnown(STRATEGIC_LABELS, 'include'),
  showLevels: firstKnown(SHOW_LEVELS_LABELS, 'everything'),
  planTypes: known(PLAN_TYPE_LABELS),
  planCurrencies: currencyCodes,
  finance: known(FINANCE_LABELS),
  search: z.string().catch(''),
});

type Address = z.infer<typeof addressSchema>;

const NO_ADDRESS: Address = {
  ...NO_FILTERS,
  planTypes: [],
  planCurrencies: [],
  finance: [],
  search: '',
};

/** Whether the address names any All work filter. */
export function hasAddressFilters(params: URLSearchParams): boolean {
  return ADDRESS_KEYS.some((key) => params.has(key));
}

/**
 * The state that an address such as /all-work?owner=U-1003&activityStatus=declined opens (feature
 * 009's cards, D-122): those filters over `base`. Values that are not valid are ignored. An ID
 * keeps itself as its name until the filter bar's choices name it.
 */
export function fromAddress(params: URLSearchParams, base: ViewState): ViewState {
  const { year, yearFrom, yearTo, search, ...address } = readAddress(params);
  const { planTypes, planCurrencies, finance, ...lists } = address;
  const filters: Filters = {
    ...lists,
    ...(planTypes.length > 0 && { planTypes }),
    ...(planCurrencies.length > 0 && { planCurrencies }),
    ...(finance.length > 0 && { finance }),
  };
  // A range of at most five years (016 FR-034); anything else falls back to one year.
  if (yearFrom !== undefined && yearTo !== undefined && isRange(yearFrom, yearTo)) {
    return withYears({ ...base, filters, search }, yearFrom, yearTo);
  }
  const { yearFrom: _from, yearTo: _to, ...single } = base;
  void _from;
  void _to;
  return { ...single, year: year ?? yearFrom ?? base.year, filters, search };
}

function isRange(from: number, to: number): boolean {
  return to > from && to - from < MAX_YEARS;
}

function readAddress(params: URLSearchParams): Address {
  const parsed = addressSchema.safeParse({
    year: params.get('year'),
    yearFrom: params.get('yearFrom') ?? params.get('year'),
    yearTo: params.get('yearTo'),
    plans: params.getAll('plan'),
    planStatuses: params.getAll('planStatus'),
    containerStatuses: params.getAll('containerStatus'),
    activityStatuses: params.getAll('activityStatus'),
    owners: params.getAll('owner'),
    teams: params.getAll('team'),
    businessUnits: params.getAll('businessUnit'),
    activityTypes: params.getAll('activityType'),
    strategic: params.getAll('strategic'),
    showLevels: params.getAll('showLevels'),
    planTypes: params.getAll('planType'),
    planCurrencies: params.getAll('planCurrency'),
    finance: params.getAll('finance'),
    search: params.get('search'),
  });
  return parsed.success ? parsed.data : NO_ADDRESS;
}

const viewName = z
  .string()
  .refine((name) => name.trim() !== '', 'Enter a name for the view.')
  // Characters, not UTF-16 units: "é" or an emoji counts once.
  .refine((name) => [...name.trim()].length <= NAME_MAX, 'Use 60 characters or fewer.');

export const saveViewSchema = z.object({ name: viewName, isDefault: z.boolean() });

export type SaveViewValues = z.infer<typeof saveViewSchema>;

export function renameViewSchema(views: SavedView[], viewId: string) {
  return z.object({
    name: viewName.superRefine((name, context) => {
      if (viewNamed(views, name, viewId)) {
        context.addIssue({ code: 'custom', message: nameTaken(name) });
      }
    }),
  });
}

export type RenameViewValues = z.infer<ReturnType<typeof renameViewSchema>>;
