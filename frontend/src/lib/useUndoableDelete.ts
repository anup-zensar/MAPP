import { useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import { ApiError, resolvedMessage } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { useToast } from '@/ui/Toast/useToast';
import { useFormatters } from './useFormatters';

type DeleteResult = components['schemas']['DeleteResult'];
type RecycleBinEntry = components['schemas']['RecycleBinEntry'];

export interface UndoableDeleteOptions {
  /** The lists to refresh after an Undo, such as [['sample-folders']]: the record is back in them. */
  invalidate: QueryKey[];
  /**
   * What screen readers hear instead of the Undo button, which may close before they reach it:
   * where else the record can be restored. "Restore it from the recycle bin" unless given.
   */
  altText?: string;
}

/**
 * Closes the Undo message that is on screen. Only one Undo is offered at a time in the whole app
 * (FR-026, D-037), so this is shared by every page that deletes; an earlier delete stays
 * restorable from the recycle bin.
 */
let closeShownUndo: (() => void) | null = null;

/**
 * The message after a delete, with Undo (US5, FR-026, plan "Shared hooks"). Give the returned
 * afterDelete the API's answer to a delete:
 *
 *   const afterDelete = useUndoableDelete({ invalidate: [['sample-folders']] });
 *   afterDelete(await deleteFolder(id));
 *
 * It shows "Sample folder 'Ideas' moved to the recycle bin" with Undo for 10 seconds, in place of
 * any earlier Undo message. Undo puts the record back exactly as it was — or says why it cannot —
 * and refreshes the given lists.
 */
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
      // Undo puts the record back exactly as it was, so it has its name from before (FR-026).
      show({ message: `${recordTitle(entry)} restored` });
    } catch (error) {
      showRefusal(error, entry);
    } finally {
      // Refreshed whatever the answer: the record is back, or someone changed it meanwhile.
      await Promise.all(invalidate.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
    }
  }

  /** Says why the Undo did not work, in plain words (FR-020, FR-025, FR-026). */
  function showRefusal(error: unknown, entry: RecycleBinEntry) {
    const failure = error instanceof ApiError ? error : new ApiError('other');
    if (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired') {
      // The session has ended: the sign-in page says so, and nothing else needs to.
      return;
    }
    if (failure.kind === 'alreadyResolved') {
      // Who brought it back, and when, in the person's own time and date format (feature 003).
      show({ message: resolvedMessage(failure, entry.recordName, dateFormat) });
      return;
    }
    // "Restore 'Ideas' first", cannot_restore (feature 002) and the other refusals are final, with
    // the API's message; a server that cannot be reached, or failed, may work the next time.
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
      message: `${recordTitle(entry)} moved to the recycle bin`,
      // A message with an action stays 10 seconds (ToastProvider), as FR-026 asks.
      action: { label: 'Undo', altText, onClick: () => undo(entry) },
    });
  };
}

/**
 * "Sample folder 'Ideas'": the record type, with a capital letter as it starts the sentence
 * (data model §5), and the record's name.
 */
function recordTitle(entry: RecycleBinEntry): string {
  const recordType = entry.recordTypeLabel;
  return `${recordType.charAt(0).toUpperCase()}${recordType.slice(1)} '${entry.recordName}'`;
}
