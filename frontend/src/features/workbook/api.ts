import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import { downloadFromApi } from '@/api/download';
import type { components } from '@/api/generated/schema';

type Schemas = components['schemas'];
export type ImportUpload = Schemas['ImportUpload'];
export type ImportCheck = Schemas['ImportCheck'];
export type ImportDetail = Schemas['ImportDetail'];
export type ImportSummary = Schemas['ImportSummary'];
export type ImportProblem = Schemas['ImportProblem'];
export type ImportLevel = Schemas['ImportLevel'];

export function useWorkbookPlan(planId: string) {
  return useQuery({
    queryKey: ['workbook', 'plan', planId],
    queryFn: () => call(api.GET('/api/v1/plans/{planId}', { params: { path: { planId } } })),
  });
}

export function usePlanImports(planId: string) {
  return useQuery({
    queryKey: ['workbook', 'imports', planId],
    queryFn: () =>
      call(api.GET('/api/v1/plans/{planId}/imports', { params: { path: { planId } } })),
  });
}

export function useImportDetail(importId: string) {
  return useQuery({
    queryKey: ['workbook', 'import', importId],
    queryFn: () => call(api.GET('/api/v1/imports/{importId}', { params: { path: { importId } } })),
  });
}

export function useCheckImport(planId: string) {
  return useMutation({
    mutationFn: (body: ImportUpload) =>
      call(
        api.POST('/api/v1/plans/{planId}/imports/check', { params: { path: { planId } }, body }),
      ),
  });
}

export function useRunImport(planId: string) {
  const refresh = useRefreshAll();
  return useMutation({
    mutationFn: (body: ImportUpload) =>
      call(api.POST('/api/v1/plans/{planId}/imports', { params: { path: { planId } }, body })),
    onSuccess: refresh,
  });
}

export function useUndoImport(importId: string) {
  const refresh = useRefreshAll();
  return useMutation({
    mutationFn: () =>
      call(api.POST('/api/v1/imports/{importId}/undo', { params: { path: { importId } } })),
    onSettled: refresh,
  });
}

export function downloadTemplate(planId: string) {
  return downloadFromApi(
    api.GET('/api/v1/plans/{planId}/import-template', {
      params: { path: { planId } },
      parseAs: 'blob',
    }),
    'import-template.xlsx',
  );
}

export function downloadExport(planId: string) {
  return downloadFromApi(
    api.GET('/api/v1/plans/{planId}/export', { params: { path: { planId } }, parseAs: 'blob' }),
    'plan.xlsx',
  );
}

function useRefreshAll() {
  const queryClient = useQueryClient();
  return () =>
    queryClient.invalidateQueries({ predicate: ({ queryKey }) => queryKey[0] !== 'session' });
}
