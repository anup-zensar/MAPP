import { Check, ExternalLink, X } from 'lucide-react';
import { useId, useState } from 'react';
import { ApiError } from '@/api/errors';
import { Button, LinkButton } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PersonLabel } from '@/ui/PersonLabel/PersonLabel';
import { purchaseRequestHref } from '@/ui/record-links';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useFormatters } from '@/format/useFormatters';
import { ApprovalChain } from './ApprovalChain';
import { ApprovalHistory } from './ApprovalHistory';
import { PurchaseRequestDecisionDialog } from './PurchaseRequestDecisionDialog';
import { usePurchaseRequestPreview } from './purchase-request-api';
import type { PurchaseRequestApprovalPreview } from './purchase-request-api';
import {
  OPEN_PURCHASE_REQUEST,
  PR_STATUS_WORDS,
  PURCHASE_REQUEST_GONE,
} from './purchase-request-texts';
import styles from './ApprovalPreviewPanel.module.css';

export interface PurchaseRequestPreviewPanelProps {
  purchaseRequestId: string;
  onClose: () => void;
  /** After Approve or Decline here: the page moves on to the next row. */
  onDecided: () => void;
}

/**
 * The approvals page's preview of a purchase request (032 FR-029): its facts, lines and total,
 * the approval chain and the latest history; **Approve**, **Decline** and **Open purchase request**.
 */
export function PurchaseRequestPreviewPanel({
  purchaseRequestId,
  onClose,
  onDecided,
}: PurchaseRequestPreviewPanelProps) {
  const preview = usePurchaseRequestPreview(purchaseRequestId);
  const failure = preview.error instanceof ApiError ? preview.error : null;
  const refused = failure?.kind === 'forbidden';
  const gone = failure?.kind === 'notFound';
  const data = preview.data;

  return (
    <SidePanel
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={data?.purchaseRequest.name ?? 'Purchase request'}
      description={
        data
          ? `${data.purchaseRequest.id} · ${PR_STATUS_WORDS[data.purchaseRequest.status]}`
          : purchaseRequestId
      }
      footer={
        data ? (
          <PreviewFooter preview={data} onDecided={onDecided} />
        ) : (
          <Button onClick={onClose}>Close</Button>
        )
      }
    >
      {data ? (
        <PreviewBody preview={data} />
      ) : gone ? (
        <p className={styles.gone}>{PURCHASE_REQUEST_GONE}</p>
      ) : refused ? (
        <p className={styles.gone}>{failure.message}</p>
      ) : preview.isError ? (
        <ErrorState error={preview.error} onRetry={() => void preview.refetch()} />
      ) : (
        <Skeleton lines={8} height="18px" />
      )}
    </SidePanel>
  );
}

/** Decline · Approve with Approve primary; self-approval is allowed for purchase requests (FR-027). */
function PreviewFooter({
  preview,
  onDecided,
}: {
  preview: PurchaseRequestApprovalPreview;
  onDecided: () => void;
}) {
  const [deciding, setDeciding] = useState<'approve' | 'decline' | null>(null);
  const { approval } = preview;
  const [section] = approval.sections;
  return (
    <>
      {section?.actions.approve.available && (
        <Button variant="primary" icon={<Check />} onClick={() => setDeciding('approve')}>
          Approve
        </Button>
      )}
      {section?.actions.decline.available && (
        <Button icon={<X />} onClick={() => setDeciding('decline')}>
          Decline
        </Button>
      )}
      <LinkButton
        to={`${purchaseRequestHref(preview.purchaseRequest.id)}#approval`}
        icon={<ExternalLink />}
      >
        {OPEN_PURCHASE_REQUEST}
      </LinkButton>
      {deciding && section && (
        <PurchaseRequestDecisionDialog
          mode={deciding}
          approval={approval}
          section={section}
          onClose={() => setDeciding(null)}
          onDecided={onDecided}
        />
      )}
    </>
  );
}

function PreviewBody({ preview }: { preview: PurchaseRequestApprovalPreview }) {
  const format = useFormatters();
  const id = useId();
  const { purchaseRequest: pr, approval } = preview;
  const [section] = approval.sections;
  const round = section?.rounds[0];
  const reason =
    section && !section.actions.approve.available ? section.actions.approve.reason : '';

  return (
    <div className={styles.body}>
      {reason && <p className={styles.muted}>{reason}</p>}
      <section className={styles.section} aria-labelledby={`${id}-details`}>
        <h3 id={`${id}-details`} className={styles.heading}>
          Details
        </h3>
        <dl className={styles.facts}>
          <dt>Plan</dt>
          <dd>{pr.plan.name}</dd>
          <dt>Vendor</dt>
          <dd>{pr.vendor || '—'}</dd>
          <dt>Owner</dt>
          <dd>{pr.owner ? <PersonLabel person={pr.owner} /> : '—'}</dd>
          <dt>MoA approver</dt>
          <dd>{approval.moaApprover?.name ?? '—'}</dd>
          <dt>Approval workflow</dt>
          <dd>{section?.workflow?.name ?? '—'}</dd>
        </dl>
      </section>
      <section className={styles.section} aria-labelledby={`${id}-lines`}>
        <h3 id={`${id}-lines`} className={styles.heading}>
          Lines
        </h3>
        <div
          className={styles.scroll}
          role="region"
          aria-label="Lines, scrolls sideways"
          tabIndex={0}
        >
          <table className={styles.table}>
            <thead>
              <tr>
                <th scope="col">Line item</th>
                <th scope="col">HX key</th>
                <th scope="col" className={styles.amount}>
                  Amount
                </th>
              </tr>
            </thead>
            <tbody>
              {pr.lines.map((line) => (
                <tr key={line.id}>
                  <th scope="row">
                    {line.lineItem.name}
                    <span className={styles.id}>{line.lineItem.id}</span>
                  </th>
                  <td>{line.hxKey}</td>
                  <td className={styles.amount}>{format.money(line.amount)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className={styles.unassigned}>
          <span>Total</span>
          <span className={styles.amountText}>{format.money(pr.amountDisplay)}</span>
        </p>
      </section>
      {round && <ApprovalChain round={round} />}
      <ApprovalHistory rounds={section?.rounds ?? []} compact />
    </div>
  );
}
