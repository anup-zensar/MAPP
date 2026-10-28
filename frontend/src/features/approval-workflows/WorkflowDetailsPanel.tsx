import { Link } from 'react-router';
import { Pencil } from 'lucide-react';
import { ApiError } from '@/api/errors';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { ApprovalAlert, ApprovalChain, timingsText } from '@/features/approvals';
import { SubjectStatusChip } from '@/ui/SubjectStatusChip/SubjectStatusChip';
import { subjectHref } from '@/ui/subject-links';
import { localDate } from '@/format/intl';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { CanChange } from '@/ui/CanChange/CanChange';
import { workflowMoreItems } from './workflow-actions';
import type { WorkflowActions } from './workflow-actions';
import { recordsWord, usedByText, WORKFLOW_GONE } from './workflow-texts';
import { useWorkflowDetails } from './api';
import type { ApprovalWorkflowDetails } from './api';
import styles from './WorkflowDetailsPanel.module.css';

export interface WorkflowDetailsPanelProps {
  workflowId: string;
  actions: WorkflowActions;
  onClose: () => void;
}

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
      description={workflow ? usedByText(workflow) : undefined}
      footer={
        workflow ? (
          <CanChange
            permission="workflows.manage"
            otherwise={<Button onClick={onClose}>Close</Button>}
          >
            <Button variant="primary" icon={<Pencil />} onClick={() => actions.edit(workflow)}>
              Edit
            </Button>
            <ActionsMenu label={workflow.name} items={workflowMoreItems(workflow, actions)} />
          </CanChange>
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
  const records = recordsWord(workflow.objectType);
  const recordsTitle = `${records.charAt(0).toUpperCase()}${records.slice(1)}`;

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
            {`Inactive — not offered on ${records}. ${recordsTitle} that use it keep it.`}
          </p>
        )}
      </div>
      <p className={styles.muted}>{`Approves ${records}`}</p>
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

      <section className={styles.section} aria-label={`${recordsTitle} using it`}>
        <h3 className={styles.heading}>{`${recordsTitle} using it`}</h3>
        {activities.length === 0 ? (
          <p className={styles.muted}>{`No ${records} use it yet.`}</p>
        ) : (
          <ul role="list" className={styles.activities}>
            {activities.map((usage) => {
              const planId = usage.path.find((record) => record.type === 'plan')?.id;
              return (
                <li key={usage.activity.id} className={styles.activity}>
                  <div className={styles.activityMain}>
                    <Link
                      to={
                        usage.activity.type === 'activity'
                          ? recordHref('activity', usage.activity.id, planId)
                          : subjectHref('purchaseRequest', usage.activity.id)
                      }
                    >
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
                    <SubjectStatusChip status={usage.displayStatus} />
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
