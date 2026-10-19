import { useState } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { FailureMessage, FormDialog, TextField, problemOf, useToast } from '@/components';
import type { FieldProblem } from '@/components';
import { recordHref } from '@/lib/recordLinks';
import { useCopyRecord, useDestinations } from './campaignApi';
import type { DestinationList } from './campaignApi';
import type { RecordTarget } from './campaignRows';
import { DestinationChoice, DestinationNotes, DestinationsLoad } from './Destinations';
import styles from './Destinations.module.css';
import { focusOnArrival } from './focusOnArrival';
import {
  contentsSentence,
  COPY_INTO,
  COPY_RULE,
  copiesText,
  copyLabel,
  copyTitle,
  createdToast,
  NAME_OF_THE_COPY,
  openName,
  VIEW,
} from './texts';

export interface CopyDialogProps {
  /** The campaign or initiative to copy. */
  target: RecordTarget;
  onClose: () => void;
}

/**
 * **Copy** (US6#4–5, FR-026, FR-027): the copy's name — the chosen destination's suggestion
 * ("Helix launch (copy)", "(copy 2)", …) until the person types their own —, where it goes (the
 * record's own parent first, as the default), what comes along and what a copy leaves behind (R13),
 * and the budget it would over-allocate. After the copy the plan page opens with the new record
 * highlighted and a message with **View**.
 */
export function CopyDialog({ target, onClose }: CopyDialogProps) {
  const { level, id, name, counts } = target;
  const navigate = useNavigate();
  const { show } = useToast();
  const destinations = useDestinations(level, id, 'copy');
  const copy = useCopyRecord();
  // null until the person chooses: the default is the record's own parent.
  const [chosen, setChosen] = useState<string | null>(null);
  // null until the person types: the name follows the chosen destination's suggestion.
  const [typedName, setTypedName] = useState<string | null>(null);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<string | null>(null);
  const destinationField = level === 'campaign' ? 'planId' : 'campaignId';

  function choiceOf(list: DestinationList) {
    const destinationId = chosen ?? defaultChoice(list);
    const destination = list.items.find((item) => item.id === destinationId);
    return { destinationId, destination, copyName: typedName ?? destination?.suggestedName ?? '' };
  }

  async function submit(list: DestinationList) {
    const { destinationId, copyName } = choiceOf(list);
    setProblems([]);
    setFailure(null);
    try {
      const created = await copy.mutateAsync({ level, id, destinationId, name: copyName });
      onClose();
      show({
        message: createdToast(level, created.name),
        action: {
          label: VIEW,
          altText: openName(created.name),
          onClick: () => void navigate(recordHref(level, created.id, created.planId)),
        },
      });
      void navigate(`/plans/${created.planId}`, { state: { highlight: created.id } });
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      const shown = ['name', destinationField];
      const others = apiError.fields.filter((field) => !shown.includes(field.field));
      setProblems(apiError.fields);
      if (apiError.fields.length === 0) {
        setFailure(apiError.message);
      } else if (others.length > 0) {
        setFailure(others.map((field) => field.message).join(' '));
      }
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={copyTitle(level, name)}
      submitLabel={copyLabel(level)}
      onSubmit={() => destinations.data && void submit(destinations.data)}
      busy={copy.isPending}
    >
      <DestinationsLoad query={destinations}>
        {(list) => {
          const { destinationId, destination, copyName } = choiceOf(list);
          return (
            <div className={styles.stack} ref={focusOnArrival}>
              {failure && <FailureMessage failure={{ message: failure }} />}
              <TextField
                label={NAME_OF_THE_COPY}
                name="copy-name"
                marking="Required"
                maxLength={120}
                value={copyName}
                onChange={setTypedName}
                error={problemOf(problems, 'name')}
              />
              <DestinationChoice
                legend={COPY_INTO}
                items={list.items}
                value={destinationId}
                onChange={setChosen}
                error={problemOf(problems, destinationField)}
              />
              <div className={styles.text}>
                <p className={styles.muted}>{copiesText(level, contentsSentence(counts))}</p>
                <p className={styles.muted}>{COPY_RULE}</p>
              </div>
              <DestinationNotes destination={destination} />
            </div>
          );
        }}
      </DestinationsLoad>
    </FormDialog>
  );
}

/** The record's own parent, else the first destination that can take the copy. */
function defaultChoice(list: DestinationList): string {
  const current = list.items.find((item) => item.current && item.available);
  return (current ?? list.items.find((item) => item.available))?.id ?? '';
}
