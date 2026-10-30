import { Check, Undo2, X } from 'lucide-react';
import { useId, useState } from 'react';
import { ApiError } from '@/api/errors';
import {
  ApprovalAlert,
  ApprovalChain,
  ApprovalHistory,
  OverPlannedCostAlert,
  PurchaseRequestDecisionDialog,
  RefusedApprove,
  approveRefusal,
  declineBoxText,
} from '@/features/approvals';
import type {
  PurchaseRequestApproval,
  PurchaseRequestApprovalSection as Section,
} from '@/features/approvals';
import { allows } from '@/session/gate-action';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useHidesUnavailable } from '@/ui/ViewOnlyNote/view-only-context';
import { useFormatters } from '@/format/useFormatters';
import {
  APPROVAL_TITLE,
  APPROVE,
  DECLINE,
  NOT_SUBMITTED,
  RECALL,
  SECTION_TITLES,
  workflowText,
} from './approval-text';
import { usePurchaseRequestApproval } from './api';
import type { PurchaseRequest } from './api';
import { PurchaseRequestRecallDialog } from './PurchaseRequestRecallDialog';
import { SubmitPurchaseRequestButton } from './SubmitPurchaseRequestDialog';
import styles from './PurchaseRequestApprovalSection.module.css';

/**
 * A purchase request's Approval section (032 FR-022 – FR-027): a change in review first, then the
 * first approval, each with its workflow, chain, latest decline, actions and history.
 */
export function PurchaseRequestApprovalSection({
  purchaseRequest,
}: {
  purchaseRequest: PurchaseRequest;
}) {
  const approval = usePurchaseRequestApproval(purchaseRequest.id);
  const failure = approval.error instanceof ApiError ? approval.error : null;
  const notForThisPerson = failure?.kind === 'notFound' || failure?.kind === 'forbidden';

  return (
    <Card title={APPROVAL_TITLE}>
      {approval.data ? (
        <div className={styles.sections}>
          {approval.data.sections.map((section) => (
            <SectionView
              key={section.kind}
              approval={approval.data}
              section={section}
              purchaseRequest={purchaseRequest}
              titled={approval.data.sections.length > 1}
            />
          ))}
        </div>
      ) : notForThisPerson ? (
        <p className={styles.note}>{failure.message}</p>
      ) : approval.isError ? (
        <ErrorState error={approval.error} onRetry={() => void approval.refetch()} />
      ) : (
        <Skeleton lines={5} height="18px" />
      )}
    </Card>
  );
}

interface SectionViewProps {
  approval: PurchaseRequestApproval;
  section: Section;
  purchaseRequest: PurchaseRequest;
  titled: boolean;
}

type Dialog = 'approve' | 'decline' | 'recall' | null;

function SectionView({ approval, section, purchaseRequest, titled }: SectionViewProps) {
  const headingId = useId();
  const refusalId = useId();
  const { dateFormat } = useFormatters();
  const [dialog, setDialog] = useState<Dialog>(null);
  const round = section.rounds[0];
  const { actions } = section;
  const canSubmit = actions.submit.available || actions.resubmit.available;
  const refusal = approveRefusal(actions);
  // Where the View only note already says why, only the Approve refusal keeps its reason.
  const viewOnly = useHidesUnavailable();
  const reasons = [
    actions.submit,
    actions.resubmit,
    actions.approve,
    actions.decline,
    actions.recall,
  ]
    .filter((action) => !action.available && action.reason !== '')
    .map((action) => action.reason)
    .filter((reason, index, all) => all.indexOf(reason) === index)
    .filter((reason) => !viewOnly || reason === refusal);
  // Edit stays the page's one primary action for a person who may edit (design system §6.3).
  const approveVariant = allows(purchaseRequest.access.edit) ? 'secondary' : 'primary';
  const decline = section.latestDecline;

  return (
    <section className={styles.section} aria-labelledby={titled ? headingId : undefined}>
      {titled && (
        <h3 id={headingId} className={styles.heading}>
          {SECTION_TITLES[section.kind]}
        </h3>
      )}
      <p>{workflowText(section.workflow)}</p>
      {decline && (
        <ApprovalAlert tone="danger" title={declineBoxText(decline, dateFormat)}>
          {decline.comment && <p className={styles.comment}>{decline.comment}</p>}
        </ApprovalAlert>
      )}
      {round ? (
        <ApprovalChain round={round} />
      ) : (
        <div className={styles.upcoming}>
          <p className={styles.subheading}>{NOT_SUBMITTED}</p>
          {section.upcomingLevels.length > 0 && (
            <ApprovalChain upcomingLevels={section.upcomingLevels} />
          )}
        </div>
      )}
      {actions.approve.available && <OverPlannedCostAlert overages={approval.overPlannedCost} />}
      <div className={styles.actions}>
        {canSubmit && (
          <SubmitPurchaseRequestButton purchaseRequest={purchaseRequest} actions={actions} />
        )}
        {actions.approve.available && (
          <Button variant={approveVariant} icon={<Check />} onClick={() => setDialog('approve')}>
            {APPROVE}
          </Button>
        )}
        {refusal && <RefusedApprove reason={refusal} reasonId={refusalId} />}
        {actions.decline.available && (
          <Button icon={<X />} onClick={() => setDialog('decline')}>
            {DECLINE}
          </Button>
        )}
        {actions.recall.available && round && (
          <Button icon={<Undo2 />} onClick={() => setDialog('recall')}>
            {RECALL}
          </Button>
        )}
      </div>
      {reasons.map((reason) => (
        <p key={reason} id={reason === refusal ? refusalId : undefined} className={styles.reason}>
          {reason}
        </p>
      ))}
      <ApprovalHistory rounds={section.rounds} />
      {(dialog === 'approve' || dialog === 'decline') && (
        <PurchaseRequestDecisionDialog
          mode={dialog}
          approval={approval}
          section={section}
          onClose={() => setDialog(null)}
        />
      )}
      {dialog === 'recall' && round && (
        <PurchaseRequestRecallDialog
          purchaseRequestId={purchaseRequest.id}
          name={purchaseRequest.name}
          kind={section.kind}
          roundId={round.id}
          onClose={() => setDialog(null)}
        />
      )}
    </section>
  );
}
