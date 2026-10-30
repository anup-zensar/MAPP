/** What overspend control (feature 037) asks the API: Settings › Overspend control. */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';

type Schemas = components['schemas'];
export type OverspendMode = Schemas['OverspendMode'];
export type OverspendApprovalMode = Schemas['OverspendApprovalMode'];
export type OverspendControl = Schemas['OverspendControl'];
export type OverspendLimit = Schemas['OverspendLimit'];
export type OverspendConfiguration = Schemas['OverspendConfiguration'];
export type OverspendConfigurationInput = Schemas['OverspendConfigurationInput'];
export type Money = Schemas['Money'];

export const overspendKeys = { configuration: ['overspend', 'configuration'] as const };

export function useOverspendConfiguration() {
  return useQuery({
    queryKey: overspendKeys.configuration,
    queryFn: () => call(api.GET('/api/v1/overspend-configuration')),
  });
}

export function useSaveOverspendConfiguration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: OverspendConfigurationInput) =>
      call(api.PUT('/api/v1/overspend-configuration', { body })),
    onSettled: () => queryClient.invalidateQueries({ queryKey: overspendKeys.configuration }),
  });
}
