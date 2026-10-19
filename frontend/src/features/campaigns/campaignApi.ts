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

export const campaignKeys = {
  planCampaigns: (planId: string): QueryKey => ['plans', planId, 'campaigns'],
  campaign: (id: string): QueryKey => ['campaigns', id],
  initiative: (id: string): QueryKey => ['initiatives', id],
  /** Feature 004's key of the plan page (GET /api/plans/{planId}). */
  plan: (planId: string): QueryKey => ['plan', planId],
};

/**
 * Everything a change to a campaign or initiative can show in: plan lists and the campaigns list
 * (['plans']), the plan page and its summary cards (['plan']), the panels, the recycle bin and the
 * activity log. Undo refreshes the same.
 */
export const PLANNING_KEYS: QueryKey[] = [
  ['plans'],
  ['plan'],
  ['campaigns'],
  ['initiatives'],
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
    queryFn: () => call(api.GET('/api/plans/{planId}/campaigns', { params: { path: { planId } } })),
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
 * The plan (feature 004's GET /api/plans/{planId}, same key as its plan page): the forms need its
 * business unit, which decides the Shell attributes shown.
 */
export function usePlan(planId: string) {
  return useQuery({
    queryKey: campaignKeys.plan(planId),
    queryFn: () => call(api.GET('/api/plans/{planId}', { params: { path: { planId } } })),
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
      call(api.POST('/api/plans/{planId}/campaigns', { params: { path: { planId } }, body })),
    onSettled: refresh,
  });
}

export function useCreateInitiative(campaignId: string) {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: InitiativeCreate) =>
      call(
        api.POST('/api/campaigns/{campaignId}/initiatives', {
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
            api.PATCH('/api/campaigns/{campaignId}', {
              params: { path: { campaignId: id } },
              body,
            }),
          )
        : call(
            api.PATCH('/api/initiatives/{initiativeId}', {
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
        ? call(api.DELETE('/api/campaigns/{campaignId}', { params: { path: { campaignId: id } } }))
        : call(
            api.DELETE('/api/initiatives/{initiativeId}', {
              params: { path: { initiativeId: id } },
            }),
          ),
  });
}

function fetchCampaign(campaignId: string) {
  return call(api.GET('/api/campaigns/{campaignId}', { params: { path: { campaignId } } }));
}

function fetchInitiative(initiativeId: string) {
  return call(api.GET('/api/initiatives/{initiativeId}', { params: { path: { initiativeId } } }));
}

function useRefresh() {
  const queryClient = useQueryClient();
  return () => refreshPlanning(queryClient);
}
