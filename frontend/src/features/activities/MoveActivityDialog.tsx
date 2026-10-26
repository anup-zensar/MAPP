import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { ApiError, conflictMessage } from '@/api/errors';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormField } from '@/ui/FormField/FormField';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { SearchField } from '@/ui/SearchField/SearchField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import type { DateFormat } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { useActivityDestinations, useMoveActivity } from './api';
import type { Activity, ActivityDestination } from './api';
import {
  moveIntro,
  movedToast,
  moveTitle,
  NO_OTHER_INITIATIVE,
  noInitiativeMatches,
  SEARCH_INITIATIVES,
} from './activity-text';
import { DestinationList, DestinationsState } from './DestinationList';
import styles from './DestinationList.module.css';
import { focusOnArrival } from './focus-on-arrival';
import { moveSchema } from './schemas';

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
  const items = destinations.data?.items ?? [];
  const form = useForm({
    resolver: zodResolver(moveSchema(items)),
    defaultValues: { initiativeId: '', search: '' },
    ...CHECK_ON_SAVE,
  });
  const query = useWatch({ control: form.control, name: 'search' });
  const [failure, setFailure] = useState<string | null>(null);

  const submit = form.handleSubmit(async ({ initiativeId }) => {
    const target = items.find((item) => item.initiative.id === initiativeId);
    if (!target) {
      // Nowhere to go: the dialog already says so.
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
      setFailure(refusalOf(error instanceof ApiError ? error : new ApiError('other'), dateFormat));
      void destinations.refetch();
    }
  });

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={moveTitle(activity.name)}
      description={moveIntro(activity.plan.name)}
      submitLabel="Move activity"
      onSubmit={() => destinations.data && void submit()}
      busy={move.isPending}
    >
      <DestinationsState query={destinations}>
        {(loaded) => {
          if (!loaded.some((item) => !item.current)) {
            return <p className={styles.muted}>{NO_OTHER_INITIATIVE}</p>;
          }
          return (
            <div className={styles.stack} ref={focusOnArrival}>
              {failure && <FailureMessage failure={{ message: failure }} />}
              <FormField
                control={form.control}
                name="search"
                render={(field) => (
                  <SearchField
                    label={SEARCH_INITIATIVES}
                    placeholder={SEARCH_INITIATIVES}
                    value={field.value}
                    onChange={field.onChange}
                  />
                )}
              />
              <FormField
                control={form.control}
                name="initiativeId"
                render={(field) => (
                  <Destinations
                    activity={activity}
                    items={matching(loaded, query)}
                    query={query}
                    value={field.value}
                    onChange={(initiativeId) => {
                      field.onChange(initiativeId);
                      form.clearErrors('initiativeId');
                    }}
                    error={field.error}
                  />
                )}
              />
            </div>
          );
        }}
      </DestinationsState>
    </FormDialog>
  );
}

interface DestinationsProps {
  activity: Activity;
  items: ActivityDestination[];
  query: string;
  value: string;
  onChange: (initiativeId: string) => void;
  error?: string;
}

function Destinations({ activity, items, query, value, onChange, error }: DestinationsProps) {
  if (items.length === 0) {
    return <p className={styles.muted}>{noInitiativeMatches(query.trim())}</p>;
  }
  return (
    <DestinationList
      legend={moveTitle(activity.name)}
      legendHidden
      destinations={items}
      purpose="move"
      value={value}
      onChange={onChange}
      error={error}
    />
  );
}

function refusalOf(error: ApiError, dateFormat: DateFormat): string {
  if (error.kind === 'conflict') {
    return conflictMessage(error, 'activity', dateFormat);
  }
  return error.fields.length > 0
    ? error.fields.map((field) => field.message).join(' ')
    : error.message;
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
