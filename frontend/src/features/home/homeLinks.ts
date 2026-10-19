/**
 * Where Home's cards lead (feature 009 plan, "Summary cards"): the lists with the same filter.
 * Records themselves are linked through `recordHref` (lib/recordLinks.ts, D-194).
 */
import { COMING_SOON_SECTIONS } from '@/app/navigation';
import type { HomeScope } from './homeApi';

/** The Plans list with the same scope as Home (feature 004: scope=myTeams or scope=all). */
export function plansListHref(scope: HomeScope): string {
  return `/plans?scope=${scope === 'myTeams' ? 'myTeams' : 'all'}`;
}

/**
 * Where "Activities I own" leads (D-174): All work filtered to the person once All work is
 * built, until then the My activities card further down Home.
 */
export function myActivitiesHref(personId: string): string {
  const allWorkBuilt = !COMING_SOON_SECTIONS.some((section) => section.path === '/all-work');
  return allWorkBuilt ? `/all-work?owner=${encodeURIComponent(personId)}` : '#my-activities';
}

/** The Approvals page on its Waiting for me tab, with one activity's preview open when given. */
export function waitingForMeHref(activityId?: string): string {
  const params = new URLSearchParams({ tab: 'waiting' });
  if (activityId) {
    params.set('activity', activityId);
  }
  return `/approvals?${params}`;
}
