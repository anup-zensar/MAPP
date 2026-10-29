import { useLocation, useParams } from 'react-router';
import { forbiddenMessage } from '@/session/access';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { usePageTitle } from '@/ui/usePageTitle';
import { useActivity, useEditActivityForm, useNewActivityForm } from './api';
import { ActivityEditor } from './ActivityEditor';
import { ActivityFormHeader } from './ActivityFormHeader';
import { FormSkeleton, LoadFailed } from './ActivityLoadStates';

export interface ActivityFormPageProps {
  /** new: /plans/:planId/initiatives/:initiativeId/activities/new; edit: …/activities/:activityId/edit. */
  mode: 'new' | 'edit';
}

const INITIATIVE_GONE = 'This initiative no longer exists — someone may have deleted it.';
const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

export function ActivityFormPage({ mode }: ActivityFormPageProps) {
  return mode === 'new' ? <NewActivityPage /> : <EditActivityPage />;
}

function NewActivityPage() {
  const { initiativeId = '' } = useParams();
  const location = useLocation();
  usePageTitle('New activity');
  const query = useNewActivityForm(initiativeId);

  const refused = forbiddenMessage(query.error);
  if (refused) {
    return <NoAccess message={refused} />;
  }
  if (query.isPending || query.isError) {
    return (
      <>
        <ActivityFormHeader title="New activity" />
        {query.isPending ? (
          <FormSkeleton />
        ) : (
          <LoadFailed
            error={query.error}
            notFoundText={INITIATIVE_GONE}
            onRetry={() => void query.refetch()}
          />
        )}
      </>
    );
  }
  // A new location (Save and add another) starts a fresh, empty form.
  return <ActivityEditor key={location.key} data={query.data} />;
}

function EditActivityPage() {
  const { activityId = '' } = useParams();
  const activity = useActivity(activityId);
  const form = useEditActivityForm(activityId);
  usePageTitle(activity.data ? `Edit ${activity.data.name}` : 'Edit activity');

  if (activity.isSuccess && form.isSuccess) {
    return <ActivityEditor key={activity.data.id} data={form.data} activity={activity.data} />;
  }
  const refused = forbiddenMessage(form.error);
  if (refused) {
    return <NoAccess message={refused} />;
  }
  const failed = activity.isError ? activity : form.isError ? form : null;
  return (
    <>
      <ActivityFormHeader title="Edit activity" />
      {failed ? (
        <LoadFailed
          error={failed.error}
          notFoundText={ACTIVITY_GONE}
          onRetry={() => void failed.refetch()}
        />
      ) : (
        <FormSkeleton />
      )}
    </>
  );
}
