/**
 * Reading Settings from the web app (feature 002 plan, "Read hooks other features reuse"): the
 * query keys and read hooks that Settings pages and the planning forms of 004–008 share. Changes
 * live in each Settings section's own …Api.ts and invalidate the keys they affect.
 *
 * Forms read the Shell attribute form with staleTime 0 and a fresh request on every mount, so a
 * setting changed a moment ago shows on the next form (SC-008).
 */
import { useQuery } from '@tanstack/react-query';
import { api, call } from './client';
import type { components } from './schema';

export type AttributeLevel = components['schemas']['AttributeLevel'];

/** The query keys of Settings data; invalidate a prefix to refresh every page that shows it. */
export const SETTINGS_KEYS = {
  models: ['models'] as const,
  activityTypes: ['activity-types'] as const,
  shellAttributes: (level: AttributeLevel) => ['shell-attributes', level] as const,
  shellAttribute: (id: string) => ['shell-attribute', id] as const,
  shellAttributeOptions: (id: string, params: object) =>
    ['shell-attribute-options', id, params] as const,
  shellAttributeForm: (level: AttributeLevel) => ['shell-attribute-form', level] as const,
  currencies: ['currencies'] as const,
  financeBudgets: (year?: number) => ['finance-budgets', year ?? 'current'] as const,
  overview: ['settings-overview'] as const,
};

/** Every model, active and inactive, by name (GET /api/v1/models). */
export function useModels() {
  return useQuery({
    queryKey: SETTINGS_KEYS.models,
    queryFn: () => call(api.GET('/api/v1/models')),
  });
}

/** Every activity type, active and inactive, by name (GET /api/v1/activity-types). */
export function useActivityTypes() {
  return useQuery({
    queryKey: SETTINGS_KEYS.activityTypes,
    queryFn: () => call(api.GET('/api/v1/activity-types')),
  });
}

/** Every currency with its rate per 1 USD, USD first (GET /api/v1/currencies). */
export function useCurrencies() {
  return useQuery({
    queryKey: SETTINGS_KEYS.currencies,
    queryFn: () => call(api.GET('/api/v1/currencies')),
  });
}

/** The finance budgets of a year with their items, and the years to choose from (default: this year). */
export function useFinanceBudgets(year?: number) {
  return useQuery({
    queryKey: SETTINGS_KEYS.financeBudgets(year),
    queryFn: () =>
      call(api.GET('/api/v1/finance-budgets', { params: { query: year ? { year } : {} } })),
  });
}

/**
 * Everything a level's form needs to show its Shell attributes (GET /api/v1/shell-attribute-forms/
 * {level}), for ShellAttributeFields and ShellAttributeValues. Always asked again when a form
 * opens (SC-008).
 */
export function useAttributeForm(level: AttributeLevel) {
  return useQuery({
    queryKey: SETTINGS_KEYS.shellAttributeForm(level),
    queryFn: () =>
      call(api.GET('/api/v1/shell-attribute-forms/{level}', { params: { path: { level } } })),
    staleTime: 0,
    refetchOnMount: 'always',
  });
}

/** The number of active records per Settings section (GET /api/v1/settings/overview). */
export function useSettingsOverview() {
  return useQuery({
    queryKey: SETTINGS_KEYS.overview,
    queryFn: () => call(api.GET('/api/v1/settings/overview')),
  });
}
