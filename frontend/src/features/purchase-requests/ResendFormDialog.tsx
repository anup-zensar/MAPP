import { asApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { useResendPurchaseRequestForm } from './api';
import type { PurchaseRequest } from './api';
import { RESEND_FORM, RESEND_TEXT, resendTitle, resentToast } from './page-text';
import styles from './PurchaseRequestPage.module.css';

export interface ResendFormDialogProps {
  purchaseRequest: PurchaseRequest;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Resend PR form (032 FR-033): a new routing of type Resend with today's values. */
export function ResendFormDialog({ purchaseRequest, open, onOpenChange }: ResendFormDialogProps) {
  const { show } = useToast();
  const resend = useResendPurchaseRequestForm(purchaseRequest.id);

  function confirm() {
    resend.mutate(purchaseRequest.version, {
      onSuccess: () => {
        show({ message: resentToast(purchaseRequest.name) });
        onOpenChange(false);
      },
    });
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={resendTitle(purchaseRequest.name)}
      confirmLabel={RESEND_FORM}
      onConfirm={confirm}
      busy={resend.isPending}
    >
      <p>{RESEND_TEXT}</p>
      {resend.error && (
        <p role="alert" className={styles.problem}>
          {asApiError(resend.error).message}
        </p>
      )}
    </ConfirmDialog>
  );
}
