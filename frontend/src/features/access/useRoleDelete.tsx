import { useState } from 'react';
import type { ReactElement } from 'react';
import { ApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { useUndoableDelete } from '@/lib/useUndoableDelete';
import { ROLES_KEY, useDeleteRole } from './accessApi';
import type { Role } from './accessApi';

export interface RoleDelete {
  /** Delete: a role people hold says why it can't go; another asks first. */
  remove: (role: Role) => void;
  /** The question; render it once on the page. */
  dialog: ReactElement | null;
}

/** Why the role can't be deleted ("'Viewer' is given to 1 person. …"); "" when it can. */
export function deleteLock(role: Role): string {
  return role.locks.find((lock) => lock.field === 'delete')?.reason ?? '';
}

/**
 * Delete a role (US4 scenarios 4–5, FR-018): a role somebody holds, or the Administrator role,
 * can't be deleted and the message says why (the API's reason); an unused one asks "Delete role
 * 'Finance reviewer'?", goes to the recycle bin and offers Undo.
 */
export function useRoleDelete(onDeleted?: (role: Role) => void): RoleDelete {
  const { show } = useToast();
  const deleteRole = useDeleteRole();
  const afterDelete = useUndoableDelete({ invalidate: [ROLES_KEY, ['access']] });
  const [pending, setPending] = useState<Role | null>(null);

  async function confirm(role: Role) {
    if (deleteRole.isPending) {
      return;
    }
    try {
      afterDelete(await deleteRole.mutateAsync(role.id));
      onDeleted?.(role);
    } catch (error) {
      const failure = error instanceof ApiError ? error : new ApiError('other');
      if (failure.kind !== 'signInRequired' && failure.kind !== 'sessionExpired') {
        show({ message: failure.message });
      }
    } finally {
      setPending(null);
    }
  }

  return {
    remove: (role) => {
      const reason = deleteLock(role);
      if (reason) {
        show({ message: reason });
        return;
      }
      setPending(role);
    },
    dialog: pending && (
      <ConfirmDialog
        open
        onOpenChange={(open) => !open && setPending(null)}
        title={`Delete role '${pending.name}'?`}
        confirmLabel="Delete role"
        tone="danger"
        busy={deleteRole.isPending}
        onConfirm={() => void confirm(pending)}
      >
        Nobody holds it. It goes to the recycle bin; you can restore it.
      </ConfirmDialog>
    ),
  };
}
