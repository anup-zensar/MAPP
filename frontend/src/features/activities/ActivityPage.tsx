import { Pencil } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { Navigate, useLocation, useParams } from 'react-router';
import { ActionsMenu, Breadcrumb, LinkButton, PageHeader } from '@/components';
import { ApprovalSection } from '@/features/approvals/ApprovalSection';
import { ApprovalStatusAction } from '@/features/approvals/ApprovalStatusAction';
import type { ApprovalActivity } from '@/features/approvals/approvalTypes';
import { LineItemsSection } from '@/features/line-items/LineItemsSection';
import { useAttributeForm } from '@/api/settingsQueries';
import { usePageTitle } from '@/lib/usePageTitle';
import { useActivity, useEditActivityForm } from './activitiesApi';
import type { Activity } from './activitiesApi';
import { ActivityHeader } from './ActivityHeader';
import { LoadFailed, PageSkeleton } from './ActivityLoadStates';
import styles from './ActivityPage.module.css';
import {
  ActivityAttributes,
  ActivityDetails,
  ActivityGoals,
  FromFinance,
} from './ActivitySections';
import { activityEditHref, activityHref, thingsText } from './activityText';
import { ReadyToSubmitPanel } from './ReadyToSubmitPanel';
import { useActivityActions } from './useActivityActions';

const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

/**
 * An activity's page (feature 006, US2, FR-019 – FR-025), at /plans/:planId/activities/:activityId:
 * the header with status, completeness and the facts, Ready to submit, the details, then the line
 * items (feature 007, in #line-items) and the approval (feature 008, in #approval). An address
 * with another plan's ID is replaced by the right one.
 */
export function ActivityPage() {
  const { planId = '', activityId = '' } = useParams();
  const location = useLocation();
  const query = useActivity(activityId);
  usePageTitle(query.data?.name ?? 'Activity');

  if (query.isPending || query.isError) {
    return (
      <>
        <Breadcrumb items={[{ label: 'Plans', to: '/plans' }, { label: 'Activity' }]} />
        <PageHeader eyebrow="Activity" title="Activity" />
        {query.isPending ? (
          <PageSkeleton />
        ) : (
          <LoadFailed
            error={query.error}
            notFoundText={ACTIVITY_GONE}
            onRetry={() => void query.refetch()}
          />
        )}
      </>
    );
  }

  const activity = query.data;
  if (activity.plan.id !== planId) {
    return (
      <Navigate replace to={`${activityHref(activity.plan.id, activity.id)}${location.hash}`} />
    );
  }
  return (
    <ActivityView
      key={activity.id}
      activity={activity}
      openReadyFirst={location.hash === '#ready'}
    />
  );
}

interface ActivityViewProps {
  activity: Activity;
  /** The address ends in #ready (a completeness chip of the plan tree): open the panel there. */
  openReadyFirst: boolean;
}

function ActivityView({ activity, openReadyFirst }: ActivityViewProps) {
  const planId = activity.plan.id;
  // The plan's business units and model, which the activity itself does not carry.
  const context = useEditActivityForm(activity.id);
  const attributeForm = useAttributeForm('activity');
  const [panelOpen, setPanelOpen] = useState(activity.readiness.problemCount > 0 || openReadyFirst);
  // How many things Submit for approval found missing; null before anyone tried.
  const [submitProblems, setSubmitProblems] = useState<number | null>(null);
  const panel = useRef<HTMLElement>(null);
  // Each request to show the panel moves the focus there once it is open.
  const [focusRequests, setFocusRequests] = useState(openReadyFirst ? 1 : 0);

  useEffect(() => {
    if (focusRequests > 0) {
      panel.current?.focus();
    }
  }, [focusRequests]);

  function openReady() {
    setPanelOpen(true);
    setFocusRequests((count) => count + 1);
  }

  function showSubmitProblems(count: number) {
    setSubmitProblems(count);
    openReady();
  }

  const approvalActivity: ApprovalActivity = {
    id: activity.id,
    name: activity.name,
    status: activity.status,
    version: activity.version,
  };
  const more = useActivityActions(activity);

  return (
    <>
      <ActivityHeader
        activity={activity}
        onOpenReady={openReady}
        actions={
          <>
            <LinkButton
              to={activityEditHref(planId, activity.id)}
              variant="primary"
              icon={<Pencil />}
            >
              Edit
            </LinkButton>
            <ApprovalStatusAction activity={approvalActivity} onShowProblems={showSubmitProblems} />
            <ActionsMenu label={activity.name} items={more.items} />
          </>
        }
      />
      {submitProblems !== null && (
        <p role="alert" className={styles.alert}>
          Submitting needs {thingsText(submitProblems)} —{' '}
          <a
            href="#ready"
            onClick={(event) => {
              event.preventDefault();
              openReady();
            }}
          >
            view list
          </a>
        </p>
      )}
      <div className={styles.sections}>
        <ReadyToSubmitPanel
          ref={panel}
          activity={activity}
          open={panelOpen}
          onOpenChange={setPanelOpen}
        />
        <ActivityDetails activity={activity} />
        <ActivityAttributes
          activity={activity}
          form={attributeForm.data}
          businessUnits={context.data?.context.businessUnits.map((unit) => unit.id)}
          error={attributeForm.error ?? context.error}
          onRetry={() => {
            void attributeForm.refetch();
            void context.refetch();
          }}
        />
        <ActivityGoals activity={activity} model={context.data?.context.model} />
        <div id="line-items" className={styles.anchor}>
          <LineItemsSection activityId={activity.id} />
        </div>
        <div id="approval" className={styles.anchor}>
          <ApprovalSection activity={approvalActivity} onShowProblems={showSubmitProblems} />
        </div>
        <FromFinance />
      </div>
      {more.dialogs}
    </>
  );
}
