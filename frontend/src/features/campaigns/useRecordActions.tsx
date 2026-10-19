import { useState } from 'react';
import type { ReactNode } from 'react';
import type { ActionsMenuItem } from '@/components';
import type { RecordTarget } from './campaignRows';
import { DeleteRecordDialog } from './DeleteRecordDialog';
import { DELETE } from './texts';

export interface RecordActions {
  /** Opens the delete dialog for a campaign or initiative (FR-028). */
  askDelete: (target: RecordTarget) => void;
  /** **Delete** for a ⋯ menu: last, after a line, in red (design system §6.1). */
  deleteItem: (target: RecordTarget) => ActionsMenuItem;
  /** The dialogs these actions open; render them once, next to the rows or the panel. */
  dialogs: ReactNode;
}

/**
 * The dialogs a campaign or initiative opens from a row's ⋯ menu or a panel's More menu: Delete
 * (US5). Move and Copy (US6) join here when they are built. `onDeleted` runs after a delete, before
 * the Undo message — a panel of the deleted record closes there.
 */
export function useRecordActions({ onDeleted }: { onDeleted?: () => void } = {}): RecordActions {
  const [deleting, setDeleting] = useState<RecordTarget | null>(null);

  return {
    askDelete: setDeleting,
    deleteItem: (target) => ({
      label: DELETE,
      tone: 'danger',
      separatorBefore: true,
      onSelect: () => setDeleting(target),
    }),
    dialogs: (
      <DeleteRecordDialog
        target={deleting}
        onClose={() => setDeleting(null)}
        onDeleted={onDeleted}
      />
    ),
  };
}
