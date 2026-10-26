/**
 * What feature 005 asks the API (plan "Data and refresh"): a plan's campaigns with their
 * initiatives, one campaign, one initiative, and every change to them. Every change waits for
 * refreshPlanning before it is done, so rows, panels, the plan's summary cards and the totals row
 * show the new values together, without a reload (FR-012, FR-015).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient, QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';

export type Campaign = components['schemas']['Campaign'];
export type Initiative = components['schemas']['Initiative'];
export type PlanCampaigns = components['schemas']['PlanCampaigns'];
export type CampaignCreate = components['schemas']['CampaignCreate'];
export type CampaignUpdate = components['schemas']['CampaignUpdate'];
export type InitiativeCreate = components['schemas']['InitiativeCreate'];
export type InitiativeUpdate = components['schemas']['InitiativeUpdate'];
export type DeleteResult = components['schemas']['DeleteResult'];
export type Money = components['schemas']['Money'];
export type PersonRef = components['schemas']['PersonRef'];
export type Destination = components['schemas']['Destination'];
export type DestinationList = components['schemas']['DestinationList'];
export type MoveConsequences = components['schemas']['MoveConsequences'];

/** What a destination list is for (FR-023, FR-026). */
export type Purpose = 'move' | 'copy';

export const campaignKeys = {
  planCampaigns: (planId: string): QueryKey => ['plans', planId, 'campaigns'],
  campaign: (id: string): QueryKey => ['campaigns', id],
  initiative: (id: string): QueryKey => ['initiatives', id],
  /** Under the record's own key, so refreshPlanning asks for them again after a change. */
  destinations: (level: 'campaign' | 'initiative', id: string, purpose: Purpose): QueryKey => [
    level === 'campaign' ? 'campaigns' : 'initiatives',
    id,
    'destinations',
    purpose,
  ],
  /** Feature 004's key of the plan page (GET /api/v1/plans/{planId}). */
  plan: (planId: string): QueryKey => ['plan', planId],
};

/**
 * Everything a change to a campaign or initiative can show in: plan lists and the campaigns list
 * (['plans']), the plan page and its summary cards (['plan']), the panels, the activity rows and
 * pages below them (feature 006), the recycle bin and the activity log. Undo refreshes the same.
 */
export const PLANNING_KEYS: QueryKey[] = [
  ['plans'],
  ['plan'],
  ['campaigns'],
  ['initiatives'],
  ['plan-activities'],
  ['activity'],
  ['recycle-bin'],
  ['activity-log'],
];

/** Asks for everything a change can show in again; a change waits for it before it is done. */
export async function refreshPlanning(queryClient: QueryClient): Promise<void> {
  await Promise.all(PLANNING_KEYS.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}

/** The campaigns of a plan with their initiatives, in the API's order, and the plan's totals. */
export function usePlanCampaigns(planId: string) {
  return useQuery({
    queryKey: campaignKeys.planCampaigns(planId),
    queryFn: () =>
      call(api.GET('/api/v1/plans/{planId}/campaigns', { params: { path: { planId } } })),
  });
}

/** One campaign with its initiatives and roll-ups (its details panel). */
export function useCampaign(campaignId: string) {
  return useQuery({
    queryKey: campaignKeys.campaign(campaignId),
    queryFn: () => fetchCampaign(campaignId),
  });
}

/** One initiative with its roll-ups. */
export function useInitiative(initiativeId: string) {
  return useQuery({
    queryKey: campaignKeys.initiative(initiativeId),
    queryFn: () => fetchInitiative(initiativeId),
  });
}

/**
 * The plan (feature 004's GET /api/v1/plans/{planId}, same key as its plan page): the forms need its
 * business unit, which decides the Shell attributes shown.
 */
export function usePlan(planId: string) {
  return useQuery({
    queryKey: campaignKeys.plan(planId),
    queryFn: () => call(api.GET('/api/v1/plans/{planId}', { params: { path: { planId } } })),
  });
}

/**
 * Asks for a campaign or initiative again, never from the cache: after a conflict the edit form
 * takes the version that includes the other person's change.
 */
export function fetchLatestRecord(
  queryClient: QueryClient,
  level: 'campaign' | 'initiative',
  id: string,
): Promise<Campaign | Initiative> {
  return level === 'campaign'
    ? queryClient.fetchQuery({
        queryKey: campaignKeys.campaign(id),
        queryFn: () => fetchCampaign(id),
        staleTime: 0,
      })
    : queryClient.fetchQuery({
        queryKey: campaignKeys.initiative(id),
        queryFn: () => fetchInitiative(id),
        staleTime: 0,
      });
}

export function useCreateCampaign(planId: string) {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: CampaignCreate) =>
      call(api.POST('/api/v1/plans/{planId}/campaigns', { params: { path: { planId } }, body })),
    onSettled: refresh,
  });
}

export function useCreateInitiative(campaignId: string) {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: InitiativeCreate) =>
      call(
        api.POST('/api/v1/campaigns/{campaignId}/initiatives', {
          params: { path: { campaignId } },
          body,
        }),
      ),
    onSettled: refresh,
  });
}

/** What a save of an edit form sends: the whole form and the version it started from (FR-018). */
export type RecordUpdate = CampaignUpdate & InitiativeUpdate;

/** Saves a campaign's or an initiative's edit form. */
export function useUpdateRecord(level: 'campaign' | 'initiative', id: string) {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: RecordUpdate): Promise<Campaign | Initiative> =>
      level === 'campaign'
        ? call(
            api.PATCH('/api/v1/campaigns/{campaignId}', {
              params: { path: { campaignId: id } },
              body,
            }),
          )
        : call(
            api.PATCH('/api/v1/initiatives/{initiativeId}', {
              params: { path: { initiativeId: id } },
              body,
            }),
          ),
    onSettled: refresh,
  });
}

/** What a delete names: the level and the record's id. */
export interface DeleteTarget {
  level: 'campaign' | 'initiative';
  id: string;
}

/**
 * Moves a campaign or initiative, with everything below it, to the recycle bin (FR-028). The
 * caller refreshes (refreshPlanning) once its panel has closed, so a panel never shows its own
 * record as gone.
 */
export function useDeleteRecord() {
  return useMutation({
    mutationFn: ({ level, id }: DeleteTarget) =>
      level === 'campaign'
        ? call(
            api.DELETE('/api/v1/campaigns/{campaignId}', { params: { path: { campaignId: id } } }),
          )
        : call(
            api.DELETE('/api/v1/initiatives/{initiativeId}', {
              params: { path: { initiativeId: id } },
            }),
          ),
  });
}

/**
 * Where a campaign or initiative can be moved or copied to (FR-023, FR-026), current first for
 * copies. Never from an earlier opening of the dialog: what is available changes with every move.
 */
export function useDestinations(level: 'campaign' | 'initiative', id: string, purpose: Purpose) {
  return useQuery({
    queryKey: campaignKeys.destinations(level, id, purpose),
    queryFn: () =>
      level === 'campaign'
        ? call(
            api.GET('/api/v1/campaigns/{campaignId}/destinations', {
              params: { path: { campaignId: id }, query: { purpose } },
            }),
          )
        : call(
            api.GET('/api/v1/initiatives/{initiativeId}/destinations', {
              params: { path: { initiativeId: id }, query: { purpose } },
            }),
          ),
    staleTime: 0,
    gcTime: 0,
  });
}

/** What a move or copy names: the record, and the plan (campaign) or campaign (initiative) to go to. */
export interface DestinationTarget {
  level: 'campaign' | 'initiative';
  id: string;
  destinationId: string;
}

/**
 * Moves a campaign to another plan of its year, or an initiative to another campaign of its plan,
 * with everything below it (FR-025). Sends the version the row or panel showed.
 */
export function useMoveRecord() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({
      level,
      id,
      destinationId,
      version,
    }: DestinationTarget & { version: number }): Promise<Campaign | Initiative> =>
      level === 'campaign'
        ? call(
            api.POST('/api/v1/campaigns/{campaignId}/move', {
              params: { path: { campaignId: id } },
              body: { planId: destinationId, version },
            }),
          )
        : call(
            api.POST('/api/v1/initiatives/{initiativeId}/move', {
              params: { path: { initiativeId: id } },
              body: { campaignId: destinationId, version },
            }),
          ),
    onSettled: refresh,
  });
}

/** Copies a campaign or initiative with everything below it, under a new name (FR-027). */
export function useCopyRecord() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({
      level,
      id,
      destinationId,
      name,
    }: DestinationTarget & { name: string }): Promise<Campaign | Initiative> =>
      level === 'campaign'
        ? call(
            api.POST('/api/v1/campaigns/{campaignId}/copy', {
              params: { path: { campaignId: id } },
              body: { planId: destinationId, name },
            }),
          )
        : call(
            api.POST('/api/v1/initiatives/{initiativeId}/copy', {
              params: { path: { initiativeId: id } },
              body: { campaignId: destinationId, name },
            }),
          ),
    onSettled: refresh,
  });
}

function fetchCampaign(campaignId: string) {
  return call(api.GET('/api/v1/campaigns/{campaignId}', { params: { path: { campaignId } } }));
}

function fetchInitiative(initiativeId: string) {
  return call(
    api.GET('/api/v1/initiatives/{initiativeId}', { params: { path: { initiativeId } } }),
  );
}

function useRefresh() {
  const queryClient = useQueryClient();
  return () => refreshPlanning(queryClient);
}
