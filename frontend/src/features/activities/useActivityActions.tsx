import { useState } from 'react';
import type { ReactNode } from 'react';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { showHistory } from '@/features/activity-log';
import { gateItem } from '@/session/gate-action';
import type { Activity } from './api';
import { ActivityDeleteQuestion } from './ActivityDeleteQuestion';
import { CopyActivityDialog } from './CopyActivityDialog';
import { MoveActivityDialog } from './MoveActivityDialog';
import { SplitActivityDialog } from './SplitActivityDialog';
import { MOVE_TO } from './activity-text';
import { SPLIT_ACTIVITY } from './split-text';

/** The actions of an activity's More menu, and the dialog they open (render it once). */
export interface ActivityActions {
  items: ActionsMenuItem[];
  dialogs: ReactNode;
}

export function useActivityActions(
  activity: Activity,
  /** What View history does; by default it brings the History section into view. */
  onViewHistory: () => void = () => showHistory(),
): ActivityActions {
  const [open, setOpen] = useState<'copy' | 'move' | 'split' | 'delete' | null>(null);
  const close = () => setOpen(null);
  const access = activity.access;

  return {
    items: [
      ...gateItem({ label: 'Copy', onSelect: () => setOpen('copy') }, access?.copy),
      ...gateItem({ label: MOVE_TO, onSelect: () => setOpen('move') }, access?.move),
      ...gateItem({ label: SPLIT_ACTIVITY, onSelect: () => setOpen('split') }, access?.edit),
      { label: 'View history', onSelect: onViewHistory },
      ...gateItem(
        {
          label: 'Delete',
          tone: 'danger',
          separatorBefore: true,
          onSelect: () => setOpen('delete'),
        },
        access?.delete,
      ),
    ],
    dialogs: (
      <>
        {/* Copy and Move start afresh each time: they leave the page when they close. */}
        {open === 'copy' && <CopyActivityDialog activity={activity} onClose={close} />}
        {open === 'move' && <MoveActivityDialog activity={activity} onClose={close} />}
        {open === 'split' && <SplitActivityDialog activity={activity} onClose={close} />}
        <ActivityDeleteQuestion
          activity={activity}
          open={open === 'delete'}
          onOpenChange={(next) => setOpen(next ? 'delete' : null)}
        />
      </>
    ),
  };
}
