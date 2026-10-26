import { useState } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { useToast } from '@/ui/Toast/useToast';
import { useActivityDestinations, useCopyActivity } from './api';
import type { Activity, ActivityDestination } from './api';
import {
  activityHref,
  CHOOSE_ANOTHER_INITIATIVE,
  COPY_TO,
  copyIntro,
  copyTitle,
  createdToast,
} from './activity-text';
import { DestinationList, DestinationsState } from './DestinationList';
import styles from './DestinationList.module.css';
import { focusOnArrival } from './focus-on-arrival';

export interface CopyActivityDialogProps {
  activity: Activity;
  onClose: () => void;
}

/**
 * **More › Copy** (US5#2, FR-030): into the activity's own initiative (chosen at first) or another
 * of the plan, under the same date rule as a move. The copy — "<name> (copy)", "(copy 2)", … — has
 * every value and copies of the line items, starts as Created without approvals, comments or
 * history, and its page opens.
 */
export function CopyActivityDialog({ activity, onClose }: CopyActivityDialogProps) {
  const navigate = useNavigate();
  const { show } = useToast();
  const destinations = useActivityDestinations(activity.id);
  const copy = useCopyActivity();
  // null until the person chooses: the activity's own initiative.
  const [chosen, setChosen] = useState<string | null>(null);
  const [choiceError, setChoiceError] = useState<string | undefined>();
  const [failure, setFailure] = useState<string | null>(null);

  async function submit(items: ActivityDestination[]) {
    const initiativeId = chosen ?? defaultChoice(items);
    if (!items.some((item) => item.initiative.id === initiativeId && item.copyProblem === null)) {
      setChoiceError(CHOOSE_ANOTHER_INITIATIVE);
      return;
    }
    setFailure(null);
    try {
      const copied = await copy.mutateAsync({ id: activity.id, initiativeId });
      onClose();
      show({ message: createdToast(copied.name) });
      void navigate(activityHref(copied.plan.id, copied.id));
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      setFailure(
        apiError.fields.length > 0
          ? apiError.fields.map((field) => field.message).join(' ')
          : apiError.message,
      );
      void destinations.refetch();
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={copyTitle(activity.name)}
      submitLabel="Copy activity"
      onSubmit={() => destinations.data && void submit(destinations.data.items)}
      busy={copy.isPending}
    >
      <DestinationsState query={destinations}>
        {(items) => (
          <div className={styles.stack} ref={focusOnArrival}>
            {failure && <FailureMessage failure={{ message: failure }} />}
            <DestinationList
              legend={COPY_TO}
              destinations={items}
              purpose="copy"
              value={chosen ?? defaultChoice(items)}
              onChange={(initiativeId) => {
                setChosen(initiativeId);
                setChoiceError(undefined);
              }}
              error={choiceError}
            />
            <p className={styles.muted}>{copyIntro(activity.lineItems.count)}</p>
          </div>
        )}
      </DestinationsState>
    </FormDialog>
  );
}

/** The activity's own initiative, when the copy can go there. */
function defaultChoice(items: ActivityDestination[]): string {
  return items.find((item) => item.current && item.copyProblem === null)?.initiative.id ?? '';
}
