import { useParams } from 'react-router';
import { ApprovalSection } from '@/features/approvals/ApprovalSection';
import { ApprovalStatusAction } from '@/features/approvals/ApprovalStatusAction';
import type { ApprovalActivity } from '@/features/approvals/approvalTypes';
import { LineItemsSection } from '@/features/line-items/LineItemsSection';
import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * An activity's page (feature 006, US2, FR-019 – FR-025), at /plans/:planId/activities/:activityId
 * — STUB of the shared foundation. Feature 006 replaces this file with the real page; the route
 * stays.
 *
 * Already wired for features 007 and 008, so none of them changes another's files:
 * - 008's ApprovalStatusAction in the header (Submit for approval, Re-submit, Approve);
 * - 007's LineItemsSection in the element with id "line-items";
 * - 008's ApprovalSection in the element with id "approval".
 * Both 008 parts get the activity and 006's function that opens Ready to submit with its problems.
 */
export function ActivityPage() {
  const { planId = '', activityId = '' } = useParams();
  // Until 006 loads the activity, the approval parts get what the address says.
  const activity: ApprovalActivity = { id: activityId, name: '', status: 'created', version: 1 };
  const showProblems = () => {};

  return (
    <BuildingPage
      eyebrow="Activity"
      title="Activity"
      description={activityId}
      breadcrumb={[
        { label: 'Plans', to: '/plans' },
        { label: planId, to: `/plans/${planId}` },
        { label: activityId },
      ]}
      actions={<ApprovalStatusAction activity={activity} onShowProblems={showProblems} />}
    >
      <div id="line-items">
        <LineItemsSection activityId={activityId} />
      </div>
      <div id="approval">
        <ApprovalSection activity={activity} onShowProblems={showProblems} />
      </div>
    </BuildingPage>
  );
}
