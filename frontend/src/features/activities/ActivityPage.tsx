import { useEffect, useRef, useState } from 'react';
import { Navigate, useLocation, useParams } from 'react-router';
import { PageHeader } from '@/components';
import { HistorySection } from '@/features/activity-log/HistorySection';
import { ApprovalSection } from '@/features/approvals/ApprovalSection';
import { useActivityApproval } from '@/features/approvals/approvalsApi';
import type { ApprovalActivity } from '@/features/approvals/approvalTypes';
import { LineItemsSection } from '@/features/line-items/LineItemsSection';
import { Notice } from '@/features/line-items/Notice';
import { useAttributeForm } from '@/api/settingsQueries';
import { usePageTitle } from '@/lib/usePageTitle';
import { useActivity, useEditActivityForm } from './activitiesApi';
import type { Activity } from './activitiesApi';
import { ActivityFigures } from './ActivityFigures';
import { ActivityHeader } from './ActivityHeader';
import { LoadFailed, PageSkeleton } from './ActivityLoadStates';
import styles from './ActivityPage.module.css';
import {
  ActivityAttributes,
  ActivityDetails,
  ActivityGoals,
  FromFinance,
} from './ActivitySections';
import { ActivityStatusNotice } from './ActivityStatusNotice';
import { activityHref, thingsText } from './activityText';
import { ReadyToSubmitPanel } from './ReadyToSubmitPanel';
import { useActivityActions } from './useActivityActions';

const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

/**
 * An activity's page (feature 006, US2, FR-019 – FR-025), at /plans/:planId/activities/:activityId,
 * on the record template of design system §2.4 B: the header, the notice of a declined or waiting
 * activity, Ready to submit (while it can still be submitted), the summary cards, Details, Shell
 * attributes, Goals, then the line items (feature 007, in #line-items), the approval
 * (feature 008, in #approval) and the History (feature 011, in #history). An address ending in
 * #ready or #completeness (Home's "Show fields", feature 009) opens Ready to submit. An address
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
        <PageHeader
          breadcrumb={[{ label: 'Plans', to: '/plans' }, { label: 'Activity' }]}
          title="Activity"
        />
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
      openReadyFirst={location.hash === '#ready' || location.hash === '#completeness'}
    />
  );
}

interface ActivityViewProps {
  activity: Activity;
  /**
   * The address ends in #ready (a completeness chip of the plan tree) or #completeness (Home's
   * Needs your attention): open the panel there.
   */
  openReadyFirst: boolean;
}

function ActivityView({ activity, openReadyFirst }: ActivityViewProps) {
  // The plan's business units and model, which the activity itself does not carry.
  const context = useEditActivityForm(activity.id);
  const attributeForm = useAttributeForm('activity');
  // Who may decide and the latest decline (feature 008); shared with the Approval section.
  const approval = useActivityApproval(activity.id);
  const [panelOpen, setPanelOpen] = useState(activity.readiness.problemCount > 0 || openReadyFirst);
  // How many things Submit for approval found missing; null before anyone tried.
  const [submitProblems, setSubmitProblems] = useState<number | null>(null);
  const panel = useRef<HTMLElement>(null);
  // Each request to show the panel moves the focus there once it is open.
  const [focusRequests, setFocusRequests] = useState(openReadyFirst ? 1 : 0);
  // Ready to submit only while the activity can still be submitted (functional review P-10).
  const canSubmit = activity.status === 'created' || activity.status === 'declined';

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
        approval={approval.data}
        moreActions={more.items}
        onOpenReady={openReady}
        onShowProblems={showSubmitProblems}
      />
      <div className={styles.sections}>
        {submitProblems !== null && (
          <Notice tone="danger" role="alert">
            <p>
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
          </Notice>
        )}
        <ActivityStatusNotice activity={activity} approval={approval.data} />
        {canSubmit && (
          <ReadyToSubmitPanel
            ref={panel}
            activity={activity}
            open={panelOpen}
            onOpenChange={setPanelOpen}
          />
        )}
        <ActivityFigures activity={activity} />
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
        <HistorySection
          recordType="activity"
          recordId={activity.id}
          recordName={activity.name}
          level="activity"
        />
      </div>
      {more.dialogs}
    </>
  );
}
