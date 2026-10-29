import { ApprovalChain, SubmissionOutcomes } from '@/features/approvals';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { usePurchaseRequestSubmission } from './api';
import { workflowText } from './approval-text';
import {
  CHANGE_ROUND_STARTS,
  STAYS_RESERVED,
  SUBMIT_CHANGES,
  submitChangesTitle,
} from './form-text';
import { PreviewDialog } from './PreviewDialog';

export interface SubmitChangesDialogProps {
  purchaseRequestId: string;
  name: string;
  open: boolean;
  busy: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}

/**
 * Submit changes on an approved purchase request (038 FR-028): before anything is saved, says that a
 * change approval round starts, with its workflow and approvers as the server works them out, and
 * that the approved amounts stay reserved until it is approved.
 */
export function SubmitChangesDialog(props: SubmitChangesDialogProps) {
  const check = usePurchaseRequestSubmission(props.purchaseRequestId, props.open);
  const data = check.data;

  return (
    <PreviewDialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={submitChangesTitle(props.name)}
      confirm={{ label: SUBMIT_CHANGES, onConfirm: props.onConfirm, busy: props.busy }}
    >
      <p>{CHANGE_ROUND_STARTS}</p>
      {data ? (
        <>
          <p>{workflowText(data.workflow)}</p>
          {data.outcomes.length > 0 ? (
            <SubmissionOutcomes outcomes={data.outcomes} />
          ) : (
            data.levels.length > 0 && <ApprovalChain upcomingLevels={data.levels} />
          )}
        </>
      ) : (
        !check.isError && <Skeleton lines={2} />
      )}
      <p>{STAYS_RESERVED}</p>
    </PreviewDialog>
  );
}
