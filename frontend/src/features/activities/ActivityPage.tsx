import { useEffect, useRef, useState } from 'react';
import { Navigate, useLocation, useNavigate, useParams } from 'react-router';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Tabs } from '@/ui/Tabs/Tabs';
import { HistorySection } from '@/features/activity-log/HistorySection';
import { showHistory } from '@/features/activity-log/showHistory';
import { ApprovalSection } from '@/features/approvals/ApprovalSection';
import { useActivityApproval } from '@/features/approvals/approvalsApi';
import type { ApprovalActivity } from '@/features/approvals/approvalTypes';
import { ActivityFinancials } from '@/features/line-items/FinancialsSection';
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
import { ActivityAttributes, ActivityDetails, ActivityGoals } from './ActivitySections';
import { ActivityStatusNotice } from './ActivityStatusNotice';
import { SplitLinks } from './SplitLinks';
import { ACTIVITY_TABS, tabOf, tabSearch } from './activityTabs';
import type { ActivityTab } from './activityTabs';
import { activityHref, thingsText } from './activityText';
import { ReadyToSubmitPanel } from './ReadyToSubmitPanel';
import { useActivityActions } from './useActivityActions';

const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

/**
 * An activity's page (feature 006, US2, FR-019 – FR-025), at /plans/:planId/activities/:activityId,
 * on the record template of design system §2.4 B: the header, the notice of a declined or waiting
 * activity, the split links (016), the summary cards, then the tabs **Overview** (Ready to submit
 * while the activity can still be submitted, GSAP's Financials of feature 015 in #financials and
 * the line items of feature 007 in #line-items), **Details** (details,
 * Shell attributes, goals), **Approval** (feature 008, in #approval) and **History** (feature 011,
 * in #history). The tab is kept in the address (?tab=details); a #part opens its tab. An address
 * ending in #ready or #completeness (Home's "Show fields", feature 009) opens Ready to submit. An
 * address with another plan's ID is replaced by the right one.
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
  /**
   * The address ends in #ready (a completeness chip of the plan tree) or #completeness (Home's
   * Needs your attention): open the panel there.
   */
  openReadyFirst: boolean;
}

/** The #parts that name a place on the page to scroll to once its tab shows. */
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
