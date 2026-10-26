/**
 * What Home asks the API (feature 009 plan, "Query keys"): one query per card, so a card that
 * fails never hides the others (FR-022), and the display-currency choice of the picker.
 *
 * Every card asks again each time Home is opened or returned to (staleTime 0, refetchOnMount
 * "always"), so Home always shows current figures (FR-019).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { SESSION_KEY } from '@/app/useSession';
import type { CurrentUser } from '@/app/useSession';
import { DISPLAY_CURRENCIES_KEY } from '@/features/people';

export type HomeScope = components['schemas']['HomeScope'];
export type HomeSummary = components['schemas']['HomeSummary'];
export type HomePlanRow = components['schemas']['HomePlanRow'];
export type HomePlanPage = components['schemas']['HomePlanPage'];
export type HomeTeam = components['schemas']['HomeTeam'];
export type HomeAttentionItem = components['schemas']['HomeAttentionItem'];
export type HomeAttentionReason = components['schemas']['HomeAttentionReason'];
export type HomeAttentionPage = components['schemas']['HomeAttentionPage'];
export type WaitingItem = components['schemas']['WaitingItem'];
export type WaitingPage = components['schemas']['WaitingPage'];
export type MyActivityRow = components['schemas']['MyActivityRow'];
export type MyActivityGroup = components['schemas']['MyActivityGroup'];
export type MyActivitiesPage = components['schemas']['MyActivitiesPage'];
export type ActivityLogEntry = components['schemas']['ActivityLogEntry'];
export type RecentChangesPage = components['schemas']['RecentChangesPage'];
export type Money = components['schemas']['Money'];
export type FinancialOverview = components['schemas']['FinancialOverview'];

/** Every query of Home starts with this key; a currency change asks all of them again. */
export const HOME_KEY = ['home'] as const;

/** The query keys of the cards (plan "Query keys"). */
export const HOME_KEYS = {
  summary: (scope: HomeScope) => [...HOME_KEY, 'summary', scope] as const,
  plans: (scope: HomeScope) => [...HOME_KEY, 'plans', scope] as const,
  attention: [...HOME_KEY, 'attention'] as const,
  waiting: [...HOME_KEY, 'waiting'] as const,
  myActivities: [...HOME_KEY, 'my-activities'] as const,
  recentChanges: (scope: HomeScope) => [...HOME_KEY, 'recent-changes', scope] as const,
};

/** Current figures each time Home is opened or returned to (FR-019). */
const ALWAYS_FRESH = { staleTime: 0, refetchOnMount: 'always' } as const;

/** The four summary cards and whether to show Getting started (GET /api/v1/home/summary). */
export function useHomeSummary(scope: HomeScope) {
  return useQuery({
    queryKey: HOME_KEYS.summary(scope),
    queryFn: () => call(api.GET('/api/v1/home/summary', { params: { query: { scope } } })),
    ...ALWAYS_FRESH,
  });
}

/** Plans at a glance: at most 10 rows and the number of plans in scope (GET /api/v1/home/plans). */
export function useHomePlans(scope: HomeScope) {
  return useQuery({
    queryKey: HOME_KEYS.plans(scope),
    queryFn: () => call(api.GET('/api/v1/home/plans', { params: { query: { scope } } })),
    ...ALWAYS_FRESH,
  });
}

/** Needs your attention: every record the person owns that has a problem, in all plans. */
export function useHomeAttention() {
  return useQuery({
    queryKey: HOME_KEYS.attention,
    queryFn: () => call(api.GET('/api/v1/home/attention')),
    ...ALWAYS_FRESH,
  });
}

/** Waiting for my approval: the first 5 of 008's Waiting for me list, oldest first. */
export function useHomeWaiting() {
  return useQuery({
    queryKey: HOME_KEYS.waiting,
    queryFn: () => call(api.GET('/api/v1/home/waiting')),
    ...ALWAYS_FRESH,
  });
}

/** My activities: the activities the person owns, grouped by displayed status, all plans. */
export function useHomeMyActivities() {
  return useQuery({
    queryKey: HOME_KEYS.myActivities,
    queryFn: () => call(api.GET('/api/v1/home/my-activities')),
    ...ALWAYS_FRESH,
  });
}

/** Recent changes: the 10 newest activity-log entries (feature 011) about the plans in scope. */
export function useHomeRecentChanges(scope: HomeScope) {
  return useQuery({
    queryKey: HOME_KEYS.recentChanges(scope),
    queryFn: () => call(api.GET('/api/v1/home/recent-changes', { params: { query: { scope } } })),
    ...ALWAYS_FRESH,
  });
}

/**
 * The Financials card (015 FR-025): committed, invoiced, paid and accrued for the current year and
 * the Home scope (GET /api/v1/financials/overview). Its key starts with 'financials', so a GSAP import
 * or a link made on the Financials page asks it again (015 plan, "Query keys"); the display
 * currency is part of the key, so a new currency asks it again too.
 */
export function useFinancialOverview(scope: HomeScope, currency: string) {
  return useQuery({
    queryKey: ['financials', 'overview', scope, currency] as const,
    queryFn: () => call(api.GET('/api/v1/financials/overview', { params: { query: { scope } } })),
    ...ALWAYS_FRESH,
  });
}

/**
 * Saves the person's display currency (feature 003's preference, FR-002). The session takes the
 * saved preferences from the answer, and every card of Home asks again, so every amount is shown
 * in the new currency (SC-005). The mutation stays pending until the cards have their answers.
 */
export function useSaveDisplayCurrency() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (displayCurrency: string) =>
      call(api.PATCH('/api/v1/me/preferences', { body: { displayCurrency } })),
    onSuccess: (preferences) => {
      queryClient.setQueryData<CurrentUser | null>(SESSION_KEY, (person) =>
        person ? { ...person, preferences: { ...person.preferences, ...preferences } } : person,
      );
      // An inactive currency left behind drops off the choices (003 FR-023).
      void queryClient.invalidateQueries({ queryKey: DISPLAY_CURRENCIES_KEY });
      return queryClient.invalidateQueries({ queryKey: HOME_KEY });
    },
  });
}
