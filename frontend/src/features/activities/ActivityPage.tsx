import { useEffect, useRef, useState } from 'react';
import { Navigate, useLocation, useNavigate, useParams } from 'react-router';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Tabs } from '@/ui/Tabs/Tabs';
import { HistorySection, showHistory } from '@/features/activity-log';
import { ApprovalSection, useActivityApproval } from '@/features/approvals';
import type { ApprovalActivity } from '@/features/approvals';
import { ActivityFinancials, LineItemsSection, Notice } from '@/features/line-items';
import { ActivityPurchaseRequests } from '@/features/purchase-requests';
import { useAttributeForm } from '@/features/settings';
import { usePageTitle } from '@/ui/usePageTitle';
import { useActivity, useEditActivityForm } from './api';
import type { Activity } from './api';
import { ActivityFigures } from './ActivityFigures';
import { ActivityHeader } from './ActivityHeader';
import { LoadFailed, PageSkeleton } from './ActivityLoadStates';
import styles from './ActivityPage.module.css';
import { ActivityAttributes, ActivityDetails, ActivityGoals } from './ActivitySections';
import { ActivityStatusNotice } from './ActivityStatusNotice';
import { SplitLinks } from './SplitLinks';
import { ACTIVITY_TABS } from './activity-tabs';
import { tabOf, tabSearch } from './schemas';
import type { ActivityTab } from './schemas';
import { activityHref, thingsText } from './activity-text';
import { ReadyToSubmitPanel } from './ReadyToSubmitPanel';
import { useActivityActions } from './useActivityActions';

const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

/**
 * An activity's page (006 US2; design system §2.4 B): header, notices, summary, then the tabs
 * Overview · Details · Approval · History, kept in ?tab=; a #part opens its tab.
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
      <Navigate
        replace
        to={`${activityHref(activity.plan.id, activity.id)}${location.search}${location.hash}`}
      />
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
  /** The address ends in #ready or #completeness: Ready to submit opens first. */
  openReadyFirst: boolean;
}

const SCROLL_PARTS = new Set(['#line-items', '#financials', '#approval', '#history']);

function ActivityView({ activity, openReadyFirst }: ActivityViewProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const tab = tabOf(location.search, location.hash);
  // The plan's business units and model, which the activity itself does not carry.
  const context = useEditActivityForm(activity.id);
  const attributeForm = useAttributeForm('activity');
  // Who may decide and the latest decline (feature 008); shared with the Approval tab.
  const approval = useActivityApproval(activity.id);
  const [panelOpen, setPanelOpen] = useState(activity.readiness.problemCount > 0 || openReadyFirst);
  // How many things Submit for approval found missing; null before anyone tried.
  const [submitProblems, setSubmitProblems] = useState<number | null>(null);
  const panel = useRef<HTMLElement>(null);
  // Where the focus goes once the tab that holds it shows: Ready to submit or the History heading.
  const pendingFocus = useRef<'ready' | 'history' | null>(openReadyFirst ? 'ready' : null);
  const [focusRequests, setFocusRequests] = useState(0);
  // Ready to submit only while the activity can still be submitted (functional review P-10).
  const canSubmit = activity.status === 'created' || activity.status === 'declined';

  useEffect(() => {
    if (pendingFocus.current === 'ready' && tab === 'overview' && panel.current) {
      pendingFocus.current = null;
      panel.current.focus();
    } else if (pendingFocus.current === 'history' && tab === 'history') {
      pendingFocus.current = null;
      showHistory();
    }
  }, [focusRequests, tab]);

  // A link to #line-items, #approval … lands on its part once its tab shows.
  useEffect(() => {
    if (SCROLL_PARTS.has(location.hash)) {
      document.getElementById(location.hash.slice(1))?.scrollIntoView?.({ block: 'start' });
    }
  }, [location.hash]);

  function chooseTab(next: ActivityTab) {
    void navigate({ search: tabSearch(next), hash: '' }, { replace: true });
  }

  function openReady() {
    setPanelOpen(true);
    pendingFocus.current = 'ready';
    setFocusRequests((count) => count + 1);
    if (tab !== 'overview') {
      chooseTab('overview');
    }
  }

  function showSubmitProblems(count: number) {
    setSubmitProblems(count);
    openReady();
  }

  function viewHistory() {
    pendingFocus.current = 'history';
    setFocusRequests((count) => count + 1);
    if (tab !== 'history') {
      chooseTab('history');
    }
  }

  const approvalActivity: ApprovalActivity = {
    id: activity.id,
    name: activity.name,
    status: activity.status,
    version: activity.version,
  };
  const more = useActivityActions(activity, viewHistory);

  return (
    <>
      <ActivityHeader
        activity={activity}
        approval={approval.data}
        moreActions={more.items}
        onOpenReady={openReady}
        onShowProblems={showSubmitProblems}
      />
      <div className={styles.page}>
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
        <SplitLinks activity={activity} />
        <ActivityStatusNotice activity={activity} approval={approval.data} />
        <ActivityFigures activity={activity} />
        <Tabs label="Activity" tabs={ACTIVITY_TABS} value={tab} onChange={chooseTab}>
          {tab === 'overview' && (
            <div className={styles.sections}>
              {canSubmit && (
                <ReadyToSubmitPanel
                  ref={panel}
                  activity={activity}
                  open={panelOpen}
                  onOpenChange={setPanelOpen}
                />
              )}
              {/* GSAP's money, open, right after the summary (015 FR-015, D-305). */}
              <ActivityFinancials activityId={activity.id} />
              <div id="line-items" className={styles.anchor}>
                <LineItemsSection activityId={activity.id} />
              </div>
              <ActivityPurchaseRequests activityId={activity.id} />
            </div>
          )}
          {tab === 'details' && (
            <div className={styles.sections}>
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
            </div>
          )}
          {tab === 'approval' && (
            <div id="approval" className={styles.anchor}>
              <ApprovalSection activity={approvalActivity} onShowProblems={showSubmitProblems} />
            </div>
          )}
          {tab === 'history' && (
            <div className={styles.anchor}>
              <HistorySection
                recordType="activity"
                recordId={activity.id}
                recordName={activity.name}
                level="activity"
              />
            </div>
          )}
        </Tabs>
      </div>
      {more.dialogs}
    </>
  );
}
