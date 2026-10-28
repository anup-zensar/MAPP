import { useId } from 'react';
import { asApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { formatMoneyText } from '@/format/decimal';
import { useCreateGsapOrder } from './api';
import type { PurchaseRequest } from './api';
import {
  createText,
  GSAP_ORDER_CREATE,
  GSAP_ORDER_LINES,
  GSAP_ORDER_TITLE,
  GSAP_ORDER_UPDATE,
  GSAP_TOP_UP_TITLE,
  orderedPO,
  orderedToast,
  pendingLines,
  topUpText,
} from './gsap-order-text';
import styles from './PurchaseRequestPage.module.css';

export interface GsapOrderDialogProps {
  purchaseRequest: PurchaseRequest;
  onClose: () => void;
}

/**
 * The confirmation of a simulated GSAP run (033 US1 sc.1, US2 sc.2): what it will create, or which
 * PO lines a top-up changes and adds, with the lines it sends. A refusal stays in the dialog.
 */
export function GsapOrderDialog({ purchaseRequest, onClose }: GsapOrderDialogProps) {
  const { show } = useToast();
  const listId = useId();
  const run = useCreateGsapOrder(purchaseRequest.id);
  const topUp = orderedPO(purchaseRequest) !== '';
  const pending = pendingLines(purchaseRequest);

  function confirm() {
    run.mutate(purchaseRequest.version, {
      onSuccess: (ordered) => {
        show({ message: orderedToast(ordered) });
        onClose();
      },
    });
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={topUp ? GSAP_TOP_UP_TITLE : GSAP_ORDER_TITLE}
      confirmLabel={topUp ? GSAP_ORDER_UPDATE : GSAP_ORDER_CREATE}
      onConfirm={confirm}
      busy={run.isPending}
    >
      <div className={styles.dialogText}>
        <p>{topUp ? topUpText(purchaseRequest) : createText(purchaseRequest)}</p>
        <p id={listId} className="visually-hidden">
          {GSAP_ORDER_LINES}
        </p>
        <ul role="list" aria-labelledby={listId} className={styles.dialogList}>
          {pending.map((line) => (
            <li key={line.id}>
              {`${line.lineItem.name} (${line.lineItem.id}) · ${line.hxKey} · ${formatMoneyText(line.amount)}`}
            </li>
          ))}
        </ul>
        {run.error && (
          <p role="alert" className={styles.problem}>
            {asApiError(run.error).message}
          </p>
        )}
      </div>
    </ConfirmDialog>
  );
}
