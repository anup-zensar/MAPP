import { useState } from 'react';
import { useNavigate } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { FailureMessage, FormDialog, useToast } from '@/components';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { useDestinations, useMoveRecord } from './campaignApi';
import type { DestinationList } from './campaignApi';
import type { RecordTarget } from './campaignRows';
import { DestinationChoice, DestinationNotes, DestinationsLoad } from './Destinations';
import styles from './Destinations.module.css';
import { focusOnArrival } from './focusOnArrival';
import {
  CHOOSE_WHERE_TO_MOVE,
  MOVE_INTRO,
  MOVE_TO,
  moveLabel,
  movedToast,
  moveTitle,
  nothingToMoveTo,
  openName,
  VIEW,
} from './texts';

export interface MoveDialogProps {
  /** The campaign or initiative to move. */
  target: RecordTarget;
  onClose: () => void;
}

/**
 * **Move to…** (US6#1–3, FR-023 – FR-025): an initiative to another campaign of its plan, or a
 * campaign to another plan of its year, with everything below it. Destinations that can't take it
 * stay listed with the reason and the fix; the chosen one shows the budget it would over-allocate
 * and, for a campaign, what another model or business unit changes. After the move the plan page
 * opens with the record highlighted and a message with **View**.
 */
export function MoveDialog({ target, onClose }: MoveDialogProps) {
  const { level, id, name, year, version } = target;
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const destinations = useDestinations(level, id, 'move');
  const move = useMoveRecord();
  const [chosen, setChosen] = useState('');
  const [choiceError, setChoiceError] = useState<string | undefined>();
  const [failure, setFailure] = useState<string | null>(null);

  async function submit(list: DestinationList) {
    if (list.items.length === 0) {
      // Nothing to choose: the dialog already says so.
      return;
    }
    const destination = list.items.find((item) => item.id === chosen && item.available);
    if (!destination) {
      setChoiceError(CHOOSE_WHERE_TO_MOVE);
      return;
    }
    setFailure(null);
    try {
      const moved = await move.mutateAsync({ level, id, destinationId: destination.id, version });
      onClose();
      show({
        message: movedToast(level, moved.name, destination.name),
        action: {
          label: VIEW,
          altText: openName(moved.name),
          onClick: () => void navigate(recordHref(level, moved.id, moved.planId)),
        },
      });
      void navigate(`/plans/${moved.planId}`, { state: { highlight: moved.id } });
    } catch (error) {
      // Refused (the destination changed meanwhile) or changed by someone else: said in the
      // dialog, with the destinations asked again (refreshPlanning).
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind === 'conflict') {
        setFailure(conflictMessage(apiError, level, dateFormat));
      } else if (apiError.fields.length > 0) {
        setFailure(apiError.fields.map((field) => field.message).join(' '));
      } else {
        setFailure(apiError.message);
      }
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={moveTitle(level, name)}
      description={MOVE_INTRO[level]}
      submitLabel={moveLabel(level)}
      onSubmit={() => destinations.data && void submit(destinations.data)}
      busy={move.isPending}
    >
      <DestinationsLoad query={destinations}>
        {(list) => (
          <div className={styles.stack} ref={focusOnArrival}>
            {failure && <FailureMessage failure={{ message: failure }} />}
            <p className={styles.muted}>{list.rule}</p>
            {list.items.length === 0 ? (
              <p>{nothingToMoveTo(level, year)}</p>
            ) : (
              <>
                <DestinationChoice
                  legend={MOVE_TO}
                  legendHidden
                  items={list.items}
                  value={chosen}
                  onChange={(destinationId) => {
                    setChosen(destinationId);
                    setChoiceError(undefined);
                  }}
                  error={choiceError}
                />
                <DestinationNotes destination={list.items.find((item) => item.id === chosen)} />
              </>
            )}
          </div>
        )}
      </DestinationsLoad>
    </FormDialog>
  );
}
