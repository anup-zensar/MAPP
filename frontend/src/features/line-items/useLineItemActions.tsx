import { useState } from 'react';
import type { ReactElement } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';
import { recordHref } from '@/ui/record-links';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { DeleteLineItemDialog } from './DeleteLineItemDialog';
import { LINE_ITEM_CHANGE_KEYS, useDeleteLineItem, useDuplicateLineItem } from './api';
import type { LineItem, LineItemActivity } from './api';
import { createdToast } from './line-item-text';

interface Target {
  lineItem: LineItem;
  activity: LineItemActivity;
}

export interface LineItemActions {
  duplicate: (lineItem: LineItem, activity: LineItemActivity) => void;
  remove: (lineItem: LineItem, activity: LineItemActivity) => void;
  /** The dialogs; render them once, next to the actions. */
  dialogs: ReactElement;
}

// Duplicate and Delete of a line item; neither changes the activity's approval (D-322). `onDeleted`
// runs after a delete, before the Undo message.
export function useLineItemActions(
  onDeleted?: (activity: LineItemActivity) => void,
): LineItemActions {
  const navigate = useNavigate();
  const { show } = useToast();
  const duplicateLineItem = useDuplicateLineItem();
  const deleteLineItem = useDeleteLineItem();
  const afterDelete = useUndoableDelete({ invalidate: LINE_ITEM_CHANGE_KEYS });
  const [deleting, setDeleting] = useState<Target | null>(null);

  function showError(error: unknown) {
    show({ message: (error instanceof ApiError ? error : new ApiError('other')).message });
  }

  async function copy(lineItem: LineItem) {
    try {
      const copied = await duplicateLineItem.mutateAsync(lineItem.id);
      show({ message: createdToast(copied.name) });
      void navigate(recordHref('lineItem', copied.id));
    } catch (error) {
      showError(error);
    }
  }

  async function remove(target: Target) {
    try {
      const result = await deleteLineItem.mutateAsync(target.lineItem.id);
      setDeleting(null);
      onDeleted?.(target.activity);
      afterDelete(result);
    } catch (error) {
      setDeleting(null);
      showError(error);
    }
  }

  const dialogs = (
    <DeleteLineItemDialog
      lineItem={deleting?.lineItem ?? null}
      busy={deleteLineItem.isPending}
      onCancel={() => setDeleting(null)}
      onConfirm={() => {
        if (deleting) {
          void remove(deleting);
        }
      }}
    />
  );

  return {
    duplicate: (lineItem) => void copy(lineItem),
    remove: (lineItem, activity) => setDeleting({ lineItem, activity }),
    dialogs,
  };
}
