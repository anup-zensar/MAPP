import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import type { Control } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormField } from '@/ui/FormField/FormField';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { recordHref } from '@/ui/record-links';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { useCopyRecord, useDestinations } from './api';
import type { DestinationList } from './api';
import type { RecordTarget } from './record-target';
import { DestinationChoice, DestinationNotes, DestinationsLoad } from './Destinations';
import styles from './Destinations.module.css';
import { focusOnArrival } from './focus-on-arrival';
import { copySchema } from './schemas';
import type { CopyValues } from './schemas';
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
  const { level, id, name } = target;
  const navigate = useNavigate();
  const { show } = useToast();
  const destinations = useDestinations(level, id, 'copy');
  const copy = useCopyRecord();
  // null until the person chooses or types: the default parent and its suggested name.
  const form = useForm<CopyValues>({
    resolver: zodResolver(copySchema),
    defaultValues: { name: null, destinationId: null },
    ...CHECK_ON_SAVE,
  });
  const [failure, setFailure] = useState<string | null>(null);
  const destinationField = level === 'campaign' ? 'planId' : 'campaignId';

  async function submit(list: DestinationList, values: CopyValues) {
    const { destinationId, copyName } = choiceOf(list, values);
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
      showProblems(apiError);
    }
  }

  /** The name's and the destination's problems under them; any other one above the form. */
  function showProblems(apiError: ApiError) {
    const nameProblem = problemOf(apiError.fields, 'name');
    const destinationProblem = problemOf(apiError.fields, destinationField);
    if (nameProblem) {
      form.setError('name', { message: nameProblem });
    }
    if (destinationProblem) {
      form.setError('destinationId', { message: destinationProblem });
    }
    const shown = ['name', destinationField];
    const others = apiError.fields.filter((field) => !shown.includes(field.field));
    if (apiError.fields.length === 0) {
      setFailure(apiError.message);
    } else if (others.length > 0) {
      setFailure(others.map((field) => field.message).join(' '));
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={copyTitle(level, name)}
      submitLabel={copyLabel(level)}
      onSubmit={() => {
        const list = destinations.data;
        if (list) {
          void form.handleSubmit((values) => submit(list, values))();
        }
      }}
      busy={copy.isPending}
    >
      <DestinationsLoad query={destinations}>
        {(list) => (
          <CopyFields list={list} target={target} control={form.control} failure={failure} />
        )}
      </DestinationsLoad>
    </FormDialog>
  );
}

interface CopyFieldsProps {
  list: DestinationList;
  target: RecordTarget;
  control: Control<CopyValues>;
  failure: string | null;
}

/** The copy's name, where it goes, and what comes along, once the destinations are there. */
function CopyFields({ list, target, control, failure }: CopyFieldsProps) {
  const { level, counts } = target;
  return (
    <div className={styles.stack} ref={focusOnArrival}>
      {failure && <FailureMessage failure={{ message: failure }} />}
      <FormField
        control={control}
        name="destinationId"
        render={(destinationField) => {
          const { destinationId, destination, copyName } = choiceOf(list, {
            destinationId: destinationField.value,
          });
          return (
            <>
              <FormField
                control={control}
                name="name"
                render={(field) => (
                  <TextField
                    label={NAME_OF_THE_COPY}
                    marking="Required"
                    maxLength={120}
                    {...field}
                    name="copy-name"
                    value={field.value ?? copyName}
                  />
                )}
              />
              <DestinationChoice
                legend={COPY_INTO}
                items={list.items}
                value={destinationId}
                onChange={destinationField.onChange}
                error={destinationField.error}
              />
              <div className={styles.text}>
                <p className={styles.muted}>{copiesText(level, contentsSentence(counts))}</p>
                <p className={styles.muted}>{COPY_RULE}</p>
              </div>
              <DestinationNotes destination={destination} />
            </>
          );
        }}
      />
    </div>
  );
}

/** The chosen destination (the default one until the person chooses) and the copy's name. */
function choiceOf(list: DestinationList, values: Partial<CopyValues>) {
  const destinationId = values.destinationId ?? defaultChoice(list);
  const destination = list.items.find((item) => item.id === destinationId);
  return {
    destinationId,
    destination,
    copyName: values.name ?? destination?.suggestedName ?? '',
  };
}

/** The record's own parent, else the first destination that can take the copy. */
function defaultChoice(list: DestinationList): string {
  const current = list.items.find((item) => item.current && item.available);
  return (current ?? list.items.find((item) => item.available))?.id ?? '';
}
