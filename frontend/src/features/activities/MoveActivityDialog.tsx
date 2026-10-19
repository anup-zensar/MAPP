import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import { FailureMessage, FormDialog, SearchField, useToast } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { useActivityDestinations, useMoveActivity } from './activitiesApi';
import type { Activity, ActivityDestination } from './activitiesApi';
import {
  CHOOSE_ANOTHER_INITIATIVE,
  moveIntro,
  movedToast,
  moveTitle,
  NO_OTHER_INITIATIVE,
  noInitiativeMatches,
  SEARCH_INITIATIVES,
} from './activityText';
import { DestinationList, DestinationsState } from './DestinationList';
import styles from './DestinationList.module.css';
import { focusOnArrival } from './focusOnArrival';

export interface MoveActivityDialogProps {
  /** The activity as stored now: the move sends its version. */
  activity: Activity;
  onClose: () => void;
}

/**
 * **More › Move** (US5#1, FR-029): another initiative of the plan, searchable and grouped by
 * campaign. One whose dates don't contain the activity's, or that already has an activity with
 * its name, stays listed with the reason and what to do. The activity keeps its status, line
 * items and values; a refusal is said in the dialog.
 */
export function MoveActivityDialog({ activity, onClose }: MoveActivityDialogProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const destinations = useActivityDestinations(activity.id);
  const move = useMoveActivity();
  const [query, setQuery] = useState('');
  const [chosen, setChosen] = useState('');
  const [choiceError, setChoiceError] = useState<string | undefined>();
  const [failure, setFailure] = useState<string | null>(null);

  async function submit(items: ActivityDestination[]) {
    if (!items.some((item) => !item.current)) {
      // Nowhere to go: the dialog already says so.
      return;
    }
    const target = items.find((item) => item.initiative.id === chosen && item.moveProblem === null);
    if (!target) {
      setChoiceError(CHOOSE_ANOTHER_INITIATIVE);
      return;
    }
    setFailure(null);
    try {
      const moved = await move.mutateAsync({
        id: activity.id,
        initiativeId: target.initiative.id,
        version: activity.version,
      });
      onClose();
      show({ message: movedToast(moved.name, moved.initiative.name) });
    } catch (error) {
      // The initiative changed meanwhile, or someone else changed the activity: said here, with
      // the initiatives asked again.
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind === 'conflict') {
        setFailure(conflictMessage(apiError, 'activity', dateFormat));
      } else if (apiError.fields.length > 0) {
        setFailure(apiError.fields.map((field) => field.message).join(' '));
      } else {
        setFailure(apiError.message);
      }
      void destinations.refetch();
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={moveTitle(activity.name)}
      description={moveIntro(activity.plan.name)}
      submitLabel="Move activity"
      onSubmit={() => destinations.data && void submit(destinations.data.items)}
      busy={move.isPending}
    >
      <DestinationsState query={destinations}>
        {(items) => {
          if (!items.some((item) => !item.current)) {
            return <p className={styles.muted}>{NO_OTHER_INITIATIVE}</p>;
          }
          const shown = matching(items, query);
          return (
            <div className={styles.stack} ref={focusOnArrival}>
              {failure && <FailureMessage failure={{ message: failure }} />}
              <SearchField
                label={SEARCH_INITIATIVES}
                placeholder={SEARCH_INITIATIVES}
                value={query}
                onChange={setQuery}
              />
              {shown.length === 0 ? (
                <p className={styles.muted}>{noInitiativeMatches(query.trim())}</p>
              ) : (
                <DestinationList
                  legend={moveTitle(activity.name)}
                  legendHidden
                  destinations={shown}
                  purpose="move"
                  value={chosen}
                  onChange={(initiativeId) => {
                    setChosen(initiativeId);
                    setChoiceError(undefined);
                  }}
                  error={choiceError}
                />
              )}
            </div>
          );
        }}
      </DestinationsState>
    </FormDialog>
  );
}

/** The initiatives whose name, or whose campaign's name, contains the typed text (any case). */
function matching(items: ActivityDestination[], query: string): ActivityDestination[] {
  const text = query.trim().toLowerCase();
  if (!text) {
    return items;
  }
  return items.filter(
    (item) =>
      item.initiative.name.toLowerCase().includes(text) ||
      item.campaign.name.toLowerCase().includes(text),
  );
}
