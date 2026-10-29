import { z } from 'zod';

export const SORT_COLUMNS = [
  'name',
  'campaign',
  'owner',
  'activityType',
  'startQuarter',
  'status',
  'plannedCost',
  'committed',
  'actualSpend',
  'variancePercent',
] as const;
export type SortColumn = (typeof SORT_COLUMNS)[number];

export interface SpaSort {
  column: SortColumn;
  direction: 'asc' | 'desc';
}

/** What the address keeps (013 FR-005): the scope, the dimension, months or quarters, the sort. */
export interface SpaUrlState {
  campaign: string | undefined;
  dimension: string | undefined;
  period: 'quarter' | 'month';
  sort: SpaSort | undefined;
}

const sortSchema = z
  .string()
  .transform((text) => text.split(':'))
  .pipe(z.tuple([z.enum(SORT_COLUMNS), z.enum(['asc', 'desc'])]))
  .transform(([column, direction]) => ({ column, direction }));

const stateSchema = z.object({
  campaign: z
    .string()
    .regex(/^C-[0-9]+$/)
    .optional()
    .catch(undefined),
  dimension: z.string().min(1).max(80).optional().catch(undefined),
  period: z.enum(['quarter', 'month']).catch('quarter'),
  sort: sortSchema.optional().catch(undefined),
});

/** Reads the address; a value that is not valid falls back to its default. */
export function readSpaState(params: URLSearchParams): SpaUrlState {
  const { campaign, dimension, period, sort } = stateSchema.parse({
    campaign: params.get('campaign') ?? undefined,
    dimension: params.get('dimension') ?? undefined,
    period: params.get('period') ?? undefined,
    sort: params.get('sort') ?? undefined,
  });
  return { campaign, dimension, period, sort };
}

/** The address of a state: only what differs from the defaults. */
export function toSpaParams(state: SpaUrlState): URLSearchParams {
  const params = new URLSearchParams();
  if (state.campaign) params.set('campaign', state.campaign);
  if (state.dimension) params.set('dimension', state.dimension);
  if (state.period !== 'quarter') params.set('period', state.period);
  if (state.sort) params.set('sort', `${state.sort.column}:${state.sort.direction}`);
  return params;
}
