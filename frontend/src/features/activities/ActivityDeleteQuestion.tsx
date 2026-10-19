import { useLocation, useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { useToast } from '@/components';
import { useUndoableDelete } from '@/lib/useUndoableDelete';
import { ACTIVITY_PLANNING_KEYS, useDeleteActivity } from './activitiesApi';
import type { Activity } from './activitiesApi';
import { DeleteActivityDialog } from './DeleteActivityDialog';

export interface ActivityDeleteQuestionProps {
  activity: Activity;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * "Delete activity '<name>'?" and what follows the answer (006 plan, "Actions", FR-031): the
 * activity and its line items move to the recycle bin, the plan page opens (when it is not open
 * already) and "Activity '<name>' moved to the recycle bin" offers **Undo** for 10 seconds. A
 * refusal is said in a message. Used by the activity page's More menu and the plan tree's rows.
 */
export function ActivityDeleteQuestion({
  activity,
  open,
  onOpenChange,
}: ActivityDeleteQuestionProps) {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const { show } = useToast();
  const remove = useDeleteActivity();
  // Undo brings the activity back into every list it left.
  const afterDelete = useUndoableDelete({ invalidate: ACTIVITY_PLANNING_KEYS });

  async function confirm() {
    try {
      const result = await remove.mutateAsync(activity.id);
      onOpenChange(false);
      const planPage = `/plans/${encodeURIComponent(activity.plan.id)}`;
      if (pathname !== planPage) {
        await navigate(planPage);
      }
      afterDelete(result);
    } catch (error) {
      onOpenChange(false);
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      show({ message: apiError.message });
    }
  }

  return (
    <DeleteActivityDialog
      open={open}
      onOpenChange={onOpenChange}
      activity={activity}
      onConfirm={() => void confirm()}
      busy={remove.isPending}
    />
  );
}
