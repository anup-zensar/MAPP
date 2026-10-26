import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { ApiError, conflictMessage } from '@/api/errors';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormField } from '@/ui/FormField/FormField';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { useToast } from '@/ui/Toast/useToast';
import { recordHref } from '@/ui/record-links';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import { useDestinations, useMoveRecord } from './api';
import type { Destination } from './api';
import type { RecordTarget } from './record-target';
import { DestinationChoice, DestinationNotes, DestinationsLoad } from './Destinations';
import styles from './Destinations.module.css';
import { focusOnArrival } from './focus-on-arrival';
import { moveSchema } from './schemas';
import type { MoveValues } from './schemas';
import {
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
  const form = useForm<MoveValues>({
    resolver: zodResolver(moveSchema(destinations.data?.items ?? [])),
    defaultValues: { destinationId: '' },
    ...CHECK_ON_SAVE,
  });
  const [failure, setFailure] = useState<string | null>(null);

  async function submit(items: Destination[], { destinationId }: MoveValues) {
    const destination = items.find((item) => item.id === destinationId && item.available);
    // Nothing to choose: the dialog already says so.
    if (!destination) {
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
      onSubmit={() => {
        const list = destinations.data;
        if (list) {
          void form.handleSubmit((values) => submit(list.items, values))();
        }
      }}
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
              <FormField
                control={form.control}
                name="destinationId"
                render={(field) => (
                  <>
                    <DestinationChoice
                      legend={MOVE_TO}
                      legendHidden
                      items={list.items}
                      value={field.value}
                      onChange={(destinationId) => {
                        field.onChange(destinationId);
                        form.clearErrors('destinationId');
                      }}
                      error={field.error}
                    />
                    <DestinationNotes
                      destination={list.items.find((item) => item.id === field.value)}
                    />
                  </>
                )}
              />
            )}
          </div>
        )}
      </DestinationsLoad>
    </FormDialog>
  );
}
