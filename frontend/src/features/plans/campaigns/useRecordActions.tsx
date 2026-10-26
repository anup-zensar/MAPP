import { useState } from 'react';
import type { ReactNode } from 'react';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { gateItem } from '@/session/gate-action';
import type { RecordTarget } from './campaignRows';
import { CopyDialog } from './CopyDialog';
import { DeleteRecordDialog } from './DeleteRecordDialog';
import { MoveDialog } from './MoveDialog';
import { COPY, DELETE, MOVE_TO } from './texts';

export interface RecordActions {
  /** Opens the delete dialog for a campaign or initiative (FR-028). */
  askDelete: (target: RecordTarget) => void;
  /** Opens the copy dialog (FR-026). */
  askCopy: (target: RecordTarget) => void;
  /** Opens the move dialog (FR-023). */
  askMove: (target: RecordTarget) => void;
  /** **Copy** and **Move to…** for a ⋯ menu (US6). */
  copyAndMoveItems: (target: RecordTarget) => ActionsMenuItem[];
  /** **Delete** for a ⋯ menu: last, after a line, in red (design system §6.1); none or disabled per access. */
  deleteItem: (target: RecordTarget) => ActionsMenuItem[];
  /** The dialogs these actions open; render them once, next to the rows or the panel. */
  dialogs: ReactNode;
}

/**
 * The dialogs a campaign or initiative opens from a row's ⋯ menu or a panel's More menu: Copy and
 * Move to… (US6), and Delete (US5). `onDeleted` runs after a delete, before the Undo message — a
 * panel of the deleted record closes there. A move or copy opens the plan page itself, with the
 * record highlighted.
 */
export function useRecordActions({ onDeleted }: { onDeleted?: () => void } = {}): RecordActions {
  const [deleting, setDeleting] = useState<RecordTarget | null>(null);
  const [copying, setCopying] = useState<RecordTarget | null>(null);
  const [moving, setMoving] = useState<RecordTarget | null>(null);

  return {
    askDelete: setDeleting,
    askCopy: setCopying,
    askMove: setMoving,
    copyAndMoveItems: (target) => [
      ...gateItem({ label: COPY, onSelect: () => setCopying(target) }, target.access?.copy),
      ...gateItem({ label: MOVE_TO, onSelect: () => setMoving(target) }, target.access?.move),
    ],
    deleteItem: (target) =>
      gateItem(
        {
          label: DELETE,
          tone: 'danger',
          separatorBefore: true,
          onSelect: () => setDeleting(target),
        },
        target.access?.delete,
      ),
    dialogs: (
      <>
        {/* Each opening starts afresh: the dialogs leave the page when they close. */}
        {copying && (
          <CopyDialog key={copying.id} target={copying} onClose={() => setCopying(null)} />
        )}
        {moving && <MoveDialog key={moving.id} target={moving} onClose={() => setMoving(null)} />}
        <DeleteRecordDialog
          target={deleting}
          onClose={() => setDeleting(null)}
          onDeleted={onDeleted}
        />
      </>
    ),
  };
}
