import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormField } from '@/ui/FormField/FormField';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { useActivityDestinations, useCopyActivity } from './api';
import type { Activity } from './api';
import { activityHref, COPY_TO, copyIntro, copyTitle, createdToast } from './activity-text';
import { DestinationList, DestinationsState } from './DestinationList';
import styles from './DestinationList.module.css';
import { focusOnArrival } from './focus-on-arrival';
import { copySchema, copyTargetOf } from './schemas';

export interface CopyActivityDialogProps {
  activity: Activity;
  onClose: () => void;
}

/**
 * **More › Copy** (US5#2, FR-030): into the activity's own initiative (chosen at first) or another
 * of the plan; the copy has every value and copies of the line items, and its page opens.
 */
export function CopyActivityDialog({ activity, onClose }: CopyActivityDialogProps) {
  const navigate = useNavigate();
  const { show } = useToast();
  const destinations = useActivityDestinations(activity.id);
  const copy = useCopyActivity();
  const items = destinations.data?.items ?? [];
  // "" until the person chooses: the activity's own initiative.
  const form = useForm({
    resolver: zodResolver(copySchema(items)),
    defaultValues: { initiativeId: '' },
    ...CHECK_ON_SAVE,
  });
  const [failure, setFailure] = useState<string | null>(null);

  const submit = form.handleSubmit(async ({ initiativeId }) => {
    setFailure(null);
    try {
      const copied = await copy.mutateAsync({
        id: activity.id,
        initiativeId: copyTargetOf(initiativeId, items),
      });
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
  });

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={copyTitle(activity.name)}
      submitLabel="Copy activity"
      onSubmit={() => destinations.data && void submit()}
      busy={copy.isPending}
    >
      <DestinationsState query={destinations}>
        {(loaded) => (
          <div className={styles.stack} ref={focusOnArrival}>
            {failure && <FailureMessage failure={{ message: failure }} />}
            <FormField
              control={form.control}
              name="initiativeId"
              render={(field) => (
                <DestinationList
                  legend={COPY_TO}
                  destinations={loaded}
                  purpose="copy"
                  value={copyTargetOf(field.value, loaded)}
                  onChange={(initiativeId) => {
                    field.onChange(initiativeId);
                    form.clearErrors('initiativeId');
                  }}
                  error={field.error}
                />
              )}
            />
            <p className={styles.muted}>{copyIntro(activity.lineItems.count)}</p>
          </div>
        )}
      </DestinationsState>
    </FormDialog>
  );
}
