import { useState } from 'react';
import type { ReactElement } from 'react';
import { useToast } from '@/ui/Toast/useToast';
import { plural } from '@/format/text';
import { SettingDeleteDialog } from '../SettingDeleteDialog';
import { lockReason } from './attribute-words';
import { CHANGED_BY_ANY_CHANGE, useDeleteAttribute, useSetAttributeStatus } from './api';
import type { ShellAttribute } from './api';
import { useStatusChange } from './useStatusChange';

/** What the list and the attribute page can do with an attribute besides opening it. */
export interface AttributeActions {
  /** Set inactive (false) or Reactivate (true); answers whether it worked. */
  setActive: (attribute: ShellAttribute, active: boolean) => Promise<boolean>;
  /** Delete: asks first, or says why it can't be deleted. */
  askDelete: (attribute: ShellAttribute) => void;
  /** The delete question; render it once on the page. */
  dialog: ReactElement | null;
}

/**
 * Set inactive, Reactivate and Delete of a Shell attribute (FR-004, FR-023). A locked action
 * stays offered and says why when chosen (design system §6.6): "Business unit can't be deleted,
 * set inactive, …". The delete question is the Settings one (shared/SettingDeleteDialog).
 */
export function useAttributeActions(
  onDeleted?: (attribute: ShellAttribute) => void,
): AttributeActions {
  const { show } = useToast();
  const setStatus = useSetAttributeStatus();
  const deleteAttribute = useDeleteAttribute();
  const changeStatus = useStatusChange('Shell attribute');
  const [deleting, setDeleting] = useState<ShellAttribute | null>(null);

  async function setActive(attribute: ShellAttribute, active: boolean): Promise<boolean> {
    const locked = active ? undefined : lockReason(attribute, 'active');
    if (locked) {
      show({ message: locked });
      return false;
    }
    return changeStatus({
      name: attribute.label,
      active,
      run: () => setStatus.mutateAsync({ id: attribute.id, active, version: attribute.version }),
    });
  }

  function askDelete(attribute: ShellAttribute) {
    const locked = lockReason(attribute, 'delete');
    if (locked) {
      show({ message: locked });
      return;
    }
    setDeleting(attribute);
  }

  const dialog = deleting && (
    <SettingDeleteDialog
      kind="shellAttribute"
      name={deleting.label}
      usage={deleting.usage}
      active={deleting.active}
      deleteText={
        deleting.optionCount > 0
          ? `It moves to the recycle bin together with its ${plural(deleting.optionCount, 'option')}; you can restore it.`
          : undefined
      }
      remove={() => deleteAttribute.mutateAsync(deleting.id)}
      setInactive={() => setActive(deleting, false)}
      invalidate={CHANGED_BY_ANY_CHANGE}
      onDeleted={() => onDeleted?.(deleting)}
      onClose={() => setDeleting(null)}
    />
  );

  return { setActive, askDelete, dialog };
}
