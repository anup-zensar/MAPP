import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { fetchLatestTeam, useDeleteTeam } from './api';
import type { Team } from './api';
import { useShowFailure } from './feedback';
import { deleteTeamText, teamInUseText } from './texts';

/** The lists Undo of a team delete refreshes: the team and its memberships are back. */
const UNDO_OPTIONS = { invalidate: [['teams'], ['people'], ['session']] };

export interface DeleteTeamDialogProps {
  team: Team;
  /** Sets the team inactive instead (a used team); resolves when done. */
  setInactive: (team: Team) => Promise<void>;
  /** The team is in the recycle bin. */
  onDeleted: (team: Team) => void;
  onClose: () => void;
}

/**
 * Delete of a team (US2-6, US2-7): a team that plans use can't be deleted, so the dialog lists
 * them and offers Set team inactive; an unused team goes to the recycle bin with its memberships,
 * and the message offers Undo. When a plan started using the team meanwhile, the API refuses
 * (in_use) and the dialog shows the refreshed team's plans.
 */
export function DeleteTeamDialog({ team, setInactive, onDeleted, onClose }: DeleteTeamDialogProps) {
  const queryClient = useQueryClient();
  const deleteTeam = useDeleteTeam();
  const afterDelete = useUndoableDelete(UNDO_OPTIONS);
  const showFailure = useShowFailure('team');
  const [shown, setShown] = useState(team);
  const [busy, setBusy] = useState(false);
  const used = shown.plans.length > 0;

  async function remove() {
    setBusy(true);
    try {
      afterDelete(await deleteTeam.mutateAsync(shown.id));
      onDeleted(shown);
      onClose();
    } catch (error) {
      if (error instanceof ApiError && error.kind === 'inUse') {
        const latest = await fetchLatestTeam(queryClient, shown.id);
        if (latest && latest.plans.length > 0) {
          setShown(latest);
          setBusy(false);
          return;
        }
      }
      showFailure(error);
      onClose();
    }
  }

  async function confirmSetInactive() {
    setBusy(true);
    await setInactive(shown);
    onClose();
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={used ? `'${shown.name}' can't be deleted` : `Delete team '${shown.name}'?`}
      confirmLabel={used ? 'Set team inactive' : 'Delete team'}
      tone={used ? undefined : 'danger'}
      busy={busy}
      onConfirm={used ? confirmSetInactive : remove}
    >
      {used ? teamInUseText(shown) : deleteTeamText(shown.members.length)}
    </ConfirmDialog>
  );
}
