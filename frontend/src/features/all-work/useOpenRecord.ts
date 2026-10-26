import { useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';
import { ALL_WORK_KEYS, locateRecord } from './allWorkApi';
import type { AllWorkRow } from './allWorkTypes';
import { rowHref } from './rowHref';

/**
 * Opens a row's record (FR-016, D-123): first asks whether it still exists; then remembers where
 * the page was (`beforeOpen`) and goes there. A record someone deleted shows the API's message
 * ("This activity was deleted — it is in the recycle bin") and the view refreshes.
 */
export function useOpenRecord(beforeOpen: () => void): (row: AllWorkRow) => Promise<void> {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { show } = useToast();

  return useCallback(
    async (row: AllWorkRow) => {
      try {
        await locateRecord(row.recordId);
      } catch (error) {
        const failure = error instanceof ApiError ? error : new ApiError('other');
        if (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired') {
          // The sign-in page says so; nothing else needs to.
          return;
        }
        show({ message: failure.message });
        if (failure.kind === 'notFound') {
          await queryClient.invalidateQueries({ queryKey: ALL_WORK_KEYS.all });
        }
        return;
      }
      beforeOpen();
      await navigate(rowHref(row));
    },
    [beforeOpen, navigate, queryClient, show],
  );
}
