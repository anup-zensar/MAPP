import { useState } from 'react';
import type { ReactElement } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';
import { recordHref } from '@/lib/recordLinks';
import { useUndoableDelete } from '@/lib/useUndoableDelete';
import { DeleteLineItemDialog } from './DeleteLineItemDialog';
import { isRestartRefusal } from './lineItemFormState';
import { moneyText } from './lineItemMath';
import { LINE_ITEM_CHANGE_KEYS, useDeleteLineItem, useDuplicateLineItem } from './lineItemsApi';
import type { LineItem, LineItemActivity } from './lineItemsApi';
import { copyReason, createdToast } from './lineItemText';
import { RestartApprovalDialog } from './RestartApprovalDialog';

interface Target {
  lineItem: LineItem;
  activity: LineItemActivity;
}

export interface LineItemActions {
  /** Duplicate (FR-029): the restart dialog first on a submitted or approved activity. */
  duplicate: (lineItem: LineItem, activity: LineItemActivity) => void;
  /** Delete (FR-030): the delete dialog, then the message with Undo. */
  remove: (lineItem: LineItem, activity: LineItemActivity) => void;
  /** The dialogs; render them once, next to the actions. */
  dialogs: ReactElement;
}

/**
 * Duplicate and Delete of a line item, for its details page and the activity's Line items section
 * (FR-029 – FR-031): the copy opens after its message; a deleted line item can be brought back
 * with Undo. `onDeleted` runs after a delete, before the Undo message (the details page goes back
 * to the activity).
 */
export function useLineItemActions(
  onDeleted?: (activity: LineItemActivity) => void,
): LineItemActions {
  const navigate = useNavigate();
  const { show } = useToast();
  const duplicateLineItem = useDuplicateLineItem();
  const deleteLineItem = useDeleteLineItem();
  const afterDelete = useUndoableDelete({ invalidate: LINE_ITEM_CHANGE_KEYS });
  const [copying, setCopying] = useState<Target | null>(null);
  const [deleting, setDeleting] = useState<(Target & { restart: boolean }) | null>(null);

  function showError(error: unknown) {
    show({ message: (error instanceof ApiError ? error : new ApiError('other')).message });
  }

  async function copy(target: Target, restartApproval: boolean) {
    try {
      const copied = await duplicateLineItem.mutateAsync({
        lineItemId: target.lineItem.id,
        restartApproval,
      });
      setCopying(null);
      show({ message: createdToast(copied.name) });
      void navigate(recordHref('lineItem', copied.id));
    } catch (error) {
      if (isRestartRefusal(error)) {
        // The activity was submitted meanwhile: the person confirms first (D-146).
        setCopying(target);
        return;
      }
      setCopying(null);
      showError(error);
    }
  }

  async function remove(target: Target & { restart: boolean }) {
    try {
      const result = await deleteLineItem.mutateAsync({
        lineItemId: target.lineItem.id,
        // Sent exactly when the dialog said so (007 plan, "Dialogs").
        restartApproval: target.restart,
      });
      setDeleting(null);
      onDeleted?.(target.activity);
      afterDelete(result);
    } catch (error) {
      if (isRestartRefusal(error)) {
        setDeleting({ ...target, restart: true });
        return;
      }
      setDeleting(null);
      showError(error);
    }
  }

  const dialogs = (
    <>
      <RestartApprovalDialog
        open={copying !== null}
        onOpenChange={(open) => {
          if (!open) {
            setCopying(null);
          }
        }}
        activityName={copying?.activity.name ?? ''}
        activityStatus={copying?.activity.status ?? 'created'}
        reasons={
          copying
            ? [copyReason(copying.lineItem.name, moneyText(copying.lineItem.plannedCost))]
            : []
        }
        busy={duplicateLineItem.isPending}
        onConfirm={() => {
          if (copying) {
            void copy(copying, true);
          }
        }}
      />
      <DeleteLineItemDialog
        lineItem={deleting?.lineItem ?? null}
        restartsApproval={deleting?.restart ?? false}
        busy={deleteLineItem.isPending}
        onCancel={() => setDeleting(null)}
        onConfirm={() => {
          if (deleting) {
            void remove(deleting);
          }
        }}
      />
    </>
  );

  return {
    duplicate: (lineItem, activity) => {
      if (activity.restartsApproval) {
        setCopying({ lineItem, activity });
      } else {
        void copy({ lineItem, activity }, false);
      }
    },
    remove: (lineItem, activity) =>
      setDeleting({ lineItem, activity, restart: activity.restartsApproval }),
    dialogs,
  };
}
