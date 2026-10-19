/**
 * People and teams from the API (feature 003 plan, "Data hooks"): the lists every person and team
 * field needs, one person or team, and every change with its invalidation.
 *
 * Every change, when it is done (also when it failed), refreshes the people, the teams, the
 * session (the signed-in person's teams and name) and the history lists, so a membership change
 * shows on both sides at once (FR-012). Impact checks change nothing and refresh nothing.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient, QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';
import { SESSION_KEY } from '@/app/useSession';

export type Person = components['schemas']['Person'];
export type PersonDetails = components['schemas']['PersonDetails'];
export type PersonCreate = components['schemas']['PersonCreate'];
export type PersonUpdate = components['schemas']['PersonUpdate'];
export type Team = components['schemas']['Team'];
export type TeamCreate = components['schemas']['TeamCreate'];
export type TeamUpdate = components['schemas']['TeamUpdate'];
export type TeamColour = components['schemas']['TeamColour'];
export type PersonRef = components['schemas']['PersonRef'];
export type TeamRef = components['schemas']['TeamRef'];
export type UsageCount = components['schemas']['UsageCount'];
export type ImpactWarning = components['schemas']['ImpactWarning'];

/** The query keys of people and teams; every change invalidates both (FR-012). */
export const PEOPLE_KEY = ['people'] as const;
export const TEAMS_KEY = ['teams'] as const;

/** What every change refreshes (plan "Data hooks"). */
const CHANGED_BY_EVERY_CHANGE: QueryKey[] = [
  PEOPLE_KEY,
  TEAMS_KEY,
  SESSION_KEY,
  ['activity-log'],
  ['recycle-bin'],
];

/** Everyone, active or not, with their teams, sorted by name (GET /api/people). */
export function usePeople() {
  return useQuery({
    queryKey: PEOPLE_KEY,
    queryFn: () => call(api.GET('/api/people')),
  });
}

/** Every team, active or not, with members and plans, sorted by name (GET /api/teams). */
export function useTeams() {
  return useQuery({
    queryKey: TEAMS_KEY,
    queryFn: () => call(api.GET('/api/teams')),
  });
}

/** One person with their teams and what they own or approve (GET /api/people/{personId}). */
export function usePerson(id: string) {
  return useQuery({ queryKey: [...PEOPLE_KEY, id], queryFn: () => fetchPerson(id) });
}

/** One team with its members and the plans using it (GET /api/teams/{teamId}). */
export function useTeam(id: string) {
  return useQuery({ queryKey: [...TEAMS_KEY, id], queryFn: () => fetchTeam(id) });
}

/**
 * The person as stored now, asked again (never from the cache); after a conflict the edit form
 * takes its version. Undefined when it cannot be read.
 */
export function fetchLatestPerson(
  queryClient: QueryClient,
  id: string,
): Promise<PersonDetails | undefined> {
  return queryClient
    .fetchQuery({ queryKey: [...PEOPLE_KEY, id], queryFn: () => fetchPerson(id), staleTime: 0 })
    .catch(() => undefined);
}

/** The team as stored now, asked again (never from the cache); undefined when it cannot be read. */
export function fetchLatestTeam(queryClient: QueryClient, id: string): Promise<Team | undefined> {
  return queryClient
    .fetchQuery({ queryKey: [...TEAMS_KEY, id], queryFn: () => fetchTeam(id), staleTime: 0 })
    .catch(() => undefined);
}

/** Adds a person; they can sign in straight away with the demo password. */
export function useCreatePerson() {
  const refresh = useRefreshAfterChange();
  return useMutation({
    mutationFn: (body: PersonCreate) => call(api.POST('/api/people', { body })),
    onSettled: refresh,
  });
}

/** A person's id and the whole form, with the version the form started from. */
export interface PersonChange {
  id: string;
  body: PersonUpdate;
}

/** Saves a person's name, e-mail, job title and teams (the version comes from the form). */
export function useUpdatePerson() {
  const refresh = useRefreshAfterChange();
  return useMutation({
    mutationFn: ({ id, body }: PersonChange) =>
      call(api.PATCH('/api/people/{personId}', { params: { path: { personId: id } }, body })),
    onSettled: refresh,
  });
}

/** What saving the person's form would affect (FR-013); saves and refreshes nothing. */
export function useCheckPersonImpact() {
  return useMutation({
    mutationFn: ({ id, body }: PersonChange) =>
      call(api.POST('/api/people/{personId}/impact', { params: { path: { personId: id } }, body })),
  });
}

/** Deactivate (active false) or reactivate (active true) a record, from the version shown. */
export interface StatusChange {
  id: string;
  active: boolean;
  version: number;
}

/** Deactivates or reactivates a person (people are never deleted, FR-008). */
export function useSetPersonStatus() {
  const refresh = useRefreshAfterChange();
  return useMutation({
    mutationFn: ({ id, active, version }: StatusChange) =>
      call(
        api.POST('/api/people/{personId}/status', {
          params: { path: { personId: id } },
          body: { active, version },
        }),
      ),
    onSettled: refresh,
  });
}

/** Creates a team with its members. */
export function useCreateTeam() {
  const refresh = useRefreshAfterChange();
  return useMutation({
    mutationFn: (body: TeamCreate) => call(api.POST('/api/teams', { body })),
    onSettled: refresh,
  });
}

/** A team's id and the whole form, with the version the form started from. */
export interface TeamChange {
  id: string;
  body: TeamUpdate;
}

/** Saves a team's name, description, colour and members. */
export function useUpdateTeam() {
  const refresh = useRefreshAfterChange();
  return useMutation({
    mutationFn: ({ id, body }: TeamChange) =>
      call(api.PATCH('/api/teams/{teamId}', { params: { path: { teamId: id } }, body })),
    onSettled: refresh,
  });
}

/** What saving the team's form would affect (FR-013); saves and refreshes nothing. */
export function useCheckTeamImpact() {
  return useMutation({
    mutationFn: ({ id, body }: TeamChange) =>
      call(api.POST('/api/teams/{teamId}/impact', { params: { path: { teamId: id } }, body })),
  });
}

/** Sets a team inactive or reactivates it. */
export function useSetTeamStatus() {
  const refresh = useRefreshAfterChange();
  return useMutation({
    mutationFn: ({ id, active, version }: StatusChange) =>
      call(
        api.POST('/api/teams/{teamId}/status', {
          params: { path: { teamId: id } },
          body: { active, version },
        }),
      ),
    onSettled: refresh,
  });
}

/** Moves an unused team, with its memberships, to the recycle bin; answers the entry for Undo. */
export function useDeleteTeam() {
  const refresh = useRefreshAfterChange();
  return useMutation({
    mutationFn: (id: string) =>
      call(api.DELETE('/api/teams/{teamId}', { params: { path: { teamId: id } } })),
    onSettled: refresh,
  });
}

function fetchPerson(id: string) {
  return call(api.GET('/api/people/{personId}', { params: { path: { personId: id } } }));
}

function fetchTeam(id: string) {
  return call(api.GET('/api/teams/{teamId}', { params: { path: { teamId: id } } }));
}

/**
 * Refreshes what a change can show in. A change waits for this, so the lists already show the new
 * state when its message appears.
 */
function useRefreshAfterChange() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(
      CHANGED_BY_EVERY_CHANGE.map((queryKey) => queryClient.invalidateQueries({ queryKey })),
    );
}
