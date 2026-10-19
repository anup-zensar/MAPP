import { useParams } from 'react-router';
import { SpreadEffectNote } from '@/features/line-items/SpreadEffectNote';
import { BuildingPage } from '@/features/placeholders/BuildingPage';

export interface ActivityFormPageProps {
  /**
   * new: /plans/:planId/initiatives/:initiativeId/activities/new;
   * edit: /plans/:planId/activities/:activityId/edit (?field=<field> focuses a field).
   */
  mode: 'new' | 'edit';
}

/**
 * The activity form (feature 006, US1, US3), a full page for new and edit — STUB of the shared
 * foundation. Feature 006 replaces this file; the routes stay. The edit form already places
 * feature 007's SpreadEffectNote (under the dates, with the dates typed in the form).
 */
export function ActivityFormPage({ mode }: ActivityFormPageProps) {
  const { planId = '', activityId = '' } = useParams();
  const title = mode === 'new' ? 'New activity' : 'Edit activity';

  return (
    <BuildingPage
      eyebrow="Activity"
      title={title}
      breadcrumb={[
        { label: 'Plans', to: '/plans' },
        { label: planId, to: `/plans/${planId}` },
        { label: title },
      ]}
    >
      {mode === 'edit' && (
        <SpreadEffectNote activityId={activityId} startDate="" endDate="" ongoing={false} />
      )}
    </BuildingPage>
  );
}
