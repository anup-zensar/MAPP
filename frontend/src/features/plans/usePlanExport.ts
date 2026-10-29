import { useState } from 'react';
import { api } from '@/api/client';
import { downloadFromApi } from '@/api/download';
import { ApiError } from '@/api/errors';
import { useAccess } from '@/session/access';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { useToast } from '@/ui/Toast/useToast';

type Format = 'pptx' | 'pdf';

const PREPARING = 'Preparing the file…';

/** The plan page's Export to PowerPoint and Export to PDF, for people who hold data.export (036 FR-023). */
export function usePlanExport(planId: string): ActionsMenuItem[] {
  const canExport = useAccess().has('data.export');
  const { show } = useToast();
  const [preparing, setPreparing] = useState(false);

  async function run(format: Format) {
    setPreparing(true);
    try {
      const fileName = await downloadFromApi(
        api.GET('/api/v1/plans/{planId}/export', {
          params: { path: { planId }, query: { format } },
          parseAs: 'blob',
        }),
      );
      show({ message: `Exported '${fileName}'` });
    } catch (error) {
      const failure = error instanceof ApiError ? error : new ApiError('other');
      if (failure.kind !== 'signInRequired' && failure.kind !== 'sessionExpired') {
        show({ message: failure.message });
      }
    } finally {
      setPreparing(false);
    }
  }

  if (!canExport) {
    return [];
  }
  return [
    { label: 'Export to PowerPoint', format: 'pptx' as const },
    { label: 'Export to PDF', format: 'pdf' as const },
  ].map((item) => ({
    label: item.label,
    onSelect: () => void run(item.format),
    disabled: preparing,
    ...(preparing ? { hint: PREPARING } : {}),
  }));
}
