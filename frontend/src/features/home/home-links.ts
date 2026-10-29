/**
 * Where Home's cards lead (feature 009 plan, "Summary cards"): the lists with the same filter.
 * Records themselves are linked through `recordHref` (ui/record-links.ts, D-194).
 */
import type { HomeScope } from './api';

/** The Plans list with the same scope as Home (feature 004: scope=myTeams or scope=all). */
export function plansListHref(scope: HomeScope): string {
  return `/plans?scope=${scope === 'myTeams' ? 'myTeams' : 'all'}`;
}

/**
 * Where "Activities I own" leads (D-174): All work (feature 010) filtered to the person once, on the
 * plan years of those activities (038 FR-032); All work then drops the filter from its address (D-122).
 */
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

/** The Approvals page on its Waiting for me tab, with one activity's preview open when given. */
export function waitingForMeHref(activityId?: string): string {
  const params = new URLSearchParams({ tab: 'waiting' });
  if (activityId) {
    params.set('activity', activityId);
  }
  return `/approvals?${params}`;
}

/**
 * All work filtered to the activities committed above planned cost in the year (015 FR-024,
 * FR-025): the Financials card's first link.
 */
export function committedAboveHref(year: number): string {
  return `/all-work?finance=committedAbovePlanned&year=${year}`;
}

/** Financials on its Not linked tab (015 FR-025): the Financials card's second link. */
export const NOT_LINKED_HREF = '/financials?link=notLinked';

/** The Financials page (015). */
export const FINANCIALS_HREF = '/financials';
