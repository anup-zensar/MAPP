import type { HomeScope } from './api';

export function plansListHref(scope: HomeScope): string {
  return `/plans?scope=${scope === 'myTeams' ? 'myTeams' : 'all'}`;
}

/** All work on the plan years of the person's activities (D-174); it then drops the filter (D-122). */
export function myActivitiesHref(
  personId: string,
  yearFrom: number | null = null,
  yearTo: number | null = null,
): string {
  const params = new URLSearchParams({ owner: personId });
  if (yearFrom !== null && yearTo !== null && yearTo > yearFrom) {
    params.set('yearFrom', String(yearFrom));
    params.set('yearTo', String(yearTo));
  } else if (yearFrom !== null) {
    params.set('year', String(yearFrom));
  }
  return `/all-work?${params}`;
}

export function waitingForMeHref(activityId?: string): string {
  const params = new URLSearchParams({ tab: 'waiting' });
  if (activityId) {
    params.set('activity', activityId);
  }
  return `/approvals?${params}`;
}

export function committedAboveHref(year: number): string {
  return `/all-work?finance=committedAbovePlanned&year=${year}`;
}

/** Financials on its Not linked tab (015 FR-025): the Financials card's second link. */
export const NOT_LINKED_HREF = '/financials?link=notLinked';

export const FINANCIALS_HREF = '/financials';
