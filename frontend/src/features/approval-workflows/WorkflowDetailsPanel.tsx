import { Link } from 'react-router';
import { ApiError } from '@/api/errors';
import { Pencil } from 'lucide-react';
import { ActionsMenu, Button, ErrorState, SidePanel, Skeleton, StatusChip } from '@/components';
import { ApprovalAlert } from '@/features/approvals/ApprovalAlert';
import { ActivityStatusChip } from '@/features/activities/ActivityStatusChip';
import { ApprovalChain } from '@/features/approvals/ApprovalChain';
import { timingsText } from '@/features/approvals/messages';
import { localDate } from '@/lib/format';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { workflowMoreItems } from './workflowActions';
import type { WorkflowActions } from './workflowActions';
import { usedByText, WORKFLOW_GONE } from './workflowTexts';
import { useWorkflowDetails } from './workflowsApi';
import type { ApprovalWorkflowDetails } from './workflowsApi';
import styles from './WorkflowDetailsPanel.module.css';

export interface WorkflowDetailsPanelProps {
  workflowId: string;
  actions: WorkflowActions;
  onClose: () => void;
}

/**
 * One approval workflow (US4, FR-005), at /settings/approval-workflows/:workflowId: its status,
 * who uses it, its levels with their approvers, warnings (a level without active approver), the
 * activities that use it and who created and last changed it. The one panel footer of Settings:
 * **Edit** (primary) and ⋯ with Set inactive or Reactivate, and Delete.
 */
export function WorkflowDetailsPanel({ workflowId, actions, onClose }: WorkflowDetailsPanelProps) {
  const details = useWorkflowDetails(workflowId);
  const gone = details.error instanceof ApiError && details.error.kind === 'notFound';
  const workflow = details.data?.workflow;

  return (
    <SidePanel
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={workflow?.name ?? 'Approval workflow'}
      description={workflow ? usedByText(workflow.usedByCount, workflow.waitingCount) : undefined}
      footer={
        workflow ? (
          // The one panel footer (design system §5.3): listed primary first, as the footer lines
          // its buttons up from the right — Edit, Close, then ⋯ with Set inactive and Delete.
          <>
            <Button variant="primary" icon={<Pencil />} onClick={() => actions.edit(workflow)}>
              Edit
            </Button>
            <ActionsMenu label={workflow.name} items={workflowMoreItems(workflow, actions)} />
          </>
        ) : (
          <Button onClick={onClose}>Back to Approval workflows</Button>
        )
      }
    >
      {details.data ? (
        <DetailsBody details={details.data} />
      ) : gone ? (
        <p className={styles.muted}>{WORKFLOW_GONE}</p>
      ) : details.isError ? (
        <ErrorState error={details.error} onRetry={() => void details.refetch()} />
      ) : (
        <Skeleton lines={6} height="18px" />
      )}
    </SidePanel>
  );
}

function DetailsBody({ details }: { details: ApprovalWorkflowDetails }) {
  const format = useFormatters();
  const { workflow, activities } = details;
  const day = (timestamp: string) => format.date(localDate(timestamp));

  return (
    <div className={styles.body}>
      <div className={styles.status}>
        {workflow.active ? (
          <StatusChip tone="positive">Active</StatusChip>
        ) : (
          <StatusChip tone="neutral">Inactive</StatusChip>
        )}
        {!workflow.active && (
          <p className={styles.muted}>
            Inactive — not offered on activities. Activities that use it keep it.
          </p>
        )}
      </div>
      {workflow.description && <p className={styles.description}>{workflow.description}</p>}

      {workflow.warnings.length > 0 && (
        <ApprovalAlert tone="warning" title={workflow.warnings.join(' · ')}>
          <p>
            Anyone can still decide on their behalf. Edit the workflow to choose an active approver.
          </p>
        </ApprovalAlert>
      )}

      <section className={styles.section} aria-label="Levels">
        <h3 className={styles.heading}>Levels</h3>
        <ApprovalChain upcomingLevels={workflow.levels} />
        <p className={styles.muted}>
          {timingsText(workflow.reminderDays, workflow.escalationDays)}
        </p>
      </section>

      <section className={styles.section} aria-label="Activities using it">
        <h3 className={styles.heading}>Activities using it</h3>
        {activities.length === 0 ? (
          <p className={styles.muted}>No activities use it yet.</p>
        ) : (
          <ul role="list" className={styles.activities}>
            {activities.map((usage) => {
              const planId = usage.path.find((record) => record.type === 'plan')?.id;
              return (
                <li key={usage.activity.id} className={styles.activity}>
                  <div className={styles.activityMain}>
                    <Link to={recordHref('activity', usage.activity.id, planId)}>
                      {usage.activity.name}
                    </Link>
                    <span className={styles.secondary}>
                      {[
                        usage.activity.id,
                        ...usage.path.slice(0, 2).map((record) => record.name),
                      ].join(' · ')}
                    </span>
                  </div>
                  <div className={styles.chips}>
                    <ActivityStatusChip status={usage.displayStatus} />
                    {usage.waiting && <StatusChip tone="review">Waiting for approval</StatusChip>}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <p className={styles.secondary}>
        {`Created by ${workflow.createdBy.name} on ${day(workflow.createdAt)}, last changed by ${workflow.updatedBy.name} on ${day(workflow.updatedAt)}`}
      </p>
    </div>
  );
}
