import { useLocation, useParams } from 'react-router';
import { usePageTitle } from '@/ui/usePageTitle';
import { useActivity, useEditActivityForm, useNewActivityForm } from './api';
import { ActivityEditor } from './ActivityEditor';
import { ActivityFormHeader } from './ActivityFormHeader';
import { FormSkeleton, LoadFailed } from './ActivityLoadStates';

export interface ActivityFormPageProps {
  /**
   * new: /plans/:planId/initiatives/:initiativeId/activities/new;
   * edit: /plans/:planId/activities/:activityId/edit (?field=<field> focuses a field).
   */
  mode: 'new' | 'edit';
}

const INITIATIVE_GONE = 'This initiative no longer exists — someone may have deleted it.';
const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

/**
 * The activity form, a full page for a new and an edited activity (feature 006, US1, US3, US4;
 * 006 plan, "Activity form page").
 */
export function ActivityFormPage({ mode }: ActivityFormPageProps) {
  return mode === 'new' ? <NewActivityPage /> : <EditActivityPage />;
}

/** The new-activity page: loads the form of the initiative, then the editor. */
function NewActivityPage() {
  const { initiativeId = '' } = useParams();
  const location = useLocation();
  usePageTitle('New activity');
  const query = useNewActivityForm(initiativeId);

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

/** The edit page: loads the activity and its form, then the editor with the activity's values. */
function EditActivityPage() {
  const { activityId = '' } = useParams();
  const activity = useActivity(activityId);
  const form = useEditActivityForm(activityId);
  usePageTitle(activity.data ? `Edit ${activity.data.name}` : 'Edit activity');

  if (activity.isSuccess && form.isSuccess) {
    return <ActivityEditor key={activity.data.id} data={form.data} activity={activity.data} />;
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
