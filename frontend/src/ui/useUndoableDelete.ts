import { useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import { ApiError, resolvedMessage } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { useFormatters } from '@/format/useFormatters';
import { useToast } from './Toast/useToast';

type DeleteResult = components['schemas']['DeleteResult'];
type RecycleBinEntry = components['schemas']['RecycleBinEntry'];

export interface UndoableDeleteOptions {
  invalidate: QueryKey[];
  altText?: string;
}

// Only one Undo is offered at a time in the whole app (D-037), so this is shared by every page
// that deletes; an earlier delete stays restorable from the recycle bin.
let closeShownUndo: (() => void) | null = null;

export function useUndoableDelete({
  invalidate,
  altText = 'Restore it from the recycle bin',
}: UndoableDeleteOptions): (result: DeleteResult) => void {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const { dateFormat } = useFormatters();

  async function undo(entry: RecycleBinEntry) {
    try {
      await call(
        api.POST('/api/v1/recycle-bin/{entryId}/undo', { params: { path: { entryId: entry.id } } }),
      );
      show({ message: `${recordTitle(entry)} restored` });
    } catch (error) {
      showRefusal(error, entry);
    } finally {
      // Refreshed whatever the answer: the record is back, or someone changed it meanwhile.
      await Promise.all(invalidate.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
    }
  }

  function showRefusal(error: unknown, entry: RecycleBinEntry) {
    const failure = error instanceof ApiError ? error : new ApiError('other');
    if (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired') {
      // The session has ended: the sign-in page says so, and nothing else needs to.
      return;
    }
    if (failure.kind === 'alreadyResolved') {
      show({ message: resolvedMessage(failure, entry.recordName, dateFormat) });
      return;
    }
    const mayWorkLater = failure.kind === 'unreachable' || failure.kind === 'other';
    show({
      message: failure.message,
      action: mayWorkLater
        ? { label: 'Try again', altText: 'Try again', onClick: () => undo(entry) }
        : undefined,
    });
  }

  return function afterDelete({ recycleBinEntry: entry }: DeleteResult) {
    closeShownUndo?.();
    closeShownUndo = show({
      message: `${recordTitle(entry)} deleted`,
      action: { label: 'Undo', altText, onClick: () => undo(entry) },
    });
  };
}

function recordTitle(entry: RecycleBinEntry): string {
  const recordType = entry.recordTypeLabel;
  return `${recordType.charAt(0).toUpperCase()}${recordType.slice(1)} '${entry.recordName}'`;
}
