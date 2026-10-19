import { Link } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button, ErrorState, SidePanel, Skeleton, StatusChip } from '@/components';
import { ActivityStatusChip } from '@/features/activities/ActivityStatusChip';
import { ApprovalChain } from '@/features/approvals/ApprovalChain';
import { localDate } from '@/lib/format';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { usedByText, WORKFLOW_GONE } from './workflowTexts';
import { useWorkflowDetails } from './workflowsApi';
import type { ApprovalWorkflow, ApprovalWorkflowDetails } from './workflowsApi';
import styles from './WorkflowDetailsPanel.module.css';

export interface WorkflowActions {
  edit: (workflow: ApprovalWorkflow) => void;
  setInactive: (workflow: ApprovalWorkflow) => void;
  setActive: (workflow: ApprovalWorkflow) => void;
  remove: (workflow: ApprovalWorkflow) => void;
}

export interface WorkflowDetailsPanelProps {
  workflowId: string;
  actions: WorkflowActions;
  onClose: () => void;
}

/**
 * One approval workflow (US4, FR-005), at /settings/approval-workflows/:workflowId: its status,
 * who uses it, its levels with their approvers, warnings (a level without active approver), the
 * activities that use it and who created and last changed it; Edit, Set inactive / Set active and
 * Delete at the bottom.
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
          <>
            <Button variant="primary" onClick={() => actions.edit(workflow)}>
              Edit
            </Button>
            {workflow.active ? (
              <Button onClick={() => actions.setInactive(workflow)}>Set inactive</Button>
            ) : (
              <Button onClick={() => actions.setActive(workflow)}>Set active</Button>
            )}
            <Button
              variant="ghost"
              className={styles.delete}
              onClick={() => actions.remove(workflow)}
            >
              Delete
            </Button>
          </>
        ) : (
          <Button onClick={onClose}>Close</Button>
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
        <div className={styles.warnings}>
          {workflow.warnings.map((warning) => (
            <p key={warning} className={styles.warning}>
              {warning}
            </p>
          ))}
          <p>
            Anyone can still decide on their behalf. Edit the workflow to choose an active approver.
          </p>
        </div>
      )}

      <section className={styles.section} aria-label="Levels">
        <h3 className={styles.heading}>Levels</h3>
        <ApprovalChain upcomingLevels={workflow.levels} />
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
