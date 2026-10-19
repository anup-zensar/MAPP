import { useState } from 'react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router';
import type { ActionsMenuItem } from '@/components';
import type { Activity } from './activitiesApi';
import { ActivityDeleteQuestion } from './ActivityDeleteQuestion';

/** The actions of an activity's More menu, and the dialog they open (render it once). */
export interface ActivityActions {
  items: ActionsMenuItem[];
  dialogs: ReactNode;
}

/**
 * The More actions of an activity (006 plan, "Actions"; FR-031, FR-032): **View history**, which
 * opens the activity log filtered to the activity, and **Delete**, which asks first, moves the
 * activity and its line items to the recycle bin, opens the plan page and offers **Undo** for 10
 * seconds. Moving and copying come in a later session.
 */
export function useActivityActions(activity: Activity): ActivityActions {
  const navigate = useNavigate();
  const [asking, setAsking] = useState(false);

  return {
    items: [
      {
        label: 'View history',
        onSelect: () =>
          void navigate(
            `/activity-log?${new URLSearchParams({ recordType: 'activity', recordId: activity.id })}`,
          ),
      },
      {
        label: 'Delete',
        tone: 'danger',
        separatorBefore: true,
        onSelect: () => setAsking(true),
      },
    ],
    dialogs: <ActivityDeleteQuestion activity={activity} open={asking} onOpenChange={setAsking} />,
  };
}
