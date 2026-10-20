/**
 * Where Home's cards lead (feature 009 plan, "Summary cards"): the lists with the same filter.
 * Records themselves are linked through `recordHref` (lib/recordLinks.ts, D-194).
 */
import type { HomeScope } from './homeApi';

/** The Plans list with the same scope as Home (feature 004: scope=myTeams or scope=all). */
export function plansListHref(scope: HomeScope): string {
  return `/plans?scope=${scope === 'myTeams' ? 'myTeams' : 'all'}`;
}

/**
 * Where "Activities I own" leads (D-174): All work (feature 010) filtered to the person once; All
 * work then drops the filter from its address (D-122).
 */
export function myActivitiesHref(personId: string): string {
  return `/all-work?owner=${encodeURIComponent(personId)}`;
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
