import { useState } from 'react';
import type { ReactNode } from 'react';
import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { showHistory } from '@/features/activity-log';
import { gateItem } from '@/session/gate-action';
import type { Activity } from './activitiesApi';
import { ActivityDeleteQuestion } from './ActivityDeleteQuestion';
import { CopyActivityDialog } from './CopyActivityDialog';
import { MoveActivityDialog } from './MoveActivityDialog';
import { SplitActivityDialog } from './SplitActivityDialog';
import { SPLIT_ACTIVITY } from './splitText';

/** The actions of an activity's More menu, and the dialog they open (render it once). */
export interface ActivityActions {
  items: ActionsMenuItem[];
  dialogs: ReactNode;
}

/**
 * The More actions of an activity (006 plan, "Actions"; FR-029 – FR-032): **Copy**, **Move** and
 * **Split activity** (016 FR-026), each in its dialog; **View history**, which brings the page's History section into view;
 * and **Delete**, which asks first, moves the activity and its line items to the recycle bin,
 * opens the plan page and offers **Undo** for 10 seconds.
 */
export function useActivityActions(
  activity: Activity,
  /** What View history does; by default it brings the History section into view. */
  onViewHistory: () => void = () => showHistory(),
): ActivityActions {
  const [open, setOpen] = useState<'copy' | 'move' | 'split' | 'delete' | null>(null);
  const close = () => setOpen(null);
  // What the person may do with the activity (014 FR-011): disabled with the reason, or left out.
  const access = activity.access;

  return {
    items: [
      ...gateItem({ label: 'Copy', onSelect: () => setOpen('copy') }, access?.copy),
      ...gateItem({ label: 'Move', onSelect: () => setOpen('move') }, access?.move),
      // Split off line items and planned cost into a new activity (016 FR-026).
      ...gateItem({ label: SPLIT_ACTIVITY, onSelect: () => setOpen('split') }, access?.edit),
      // Brings the page's History into view and focuses it (feature 011, FR-011).
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
