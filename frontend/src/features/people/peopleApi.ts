/**
 * People and teams from the API (feature 003 plan, "Data hooks"). The shared foundation delivers
 * the two list reads every person and team field needs; feature 003 adds the rest here (one
 * person or team, display currencies, and every change with its invalidation).
 */
import { useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';

export type Person = components['schemas']['Person'];
export type Team = components['schemas']['Team'];
export type PersonRef = components['schemas']['PersonRef'];
export type TeamRef = components['schemas']['TeamRef'];

/** The query keys of people and teams; every change invalidates both (FR-012). */
export const PEOPLE_KEY = ['people'] as const;
export const TEAMS_KEY = ['teams'] as const;

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
