import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { asApiError, isSessionLost } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { FormField } from '@/ui/FormField/FormField';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { RECALL, REASON_LABEL, recallText, recallTitle, recalledToast } from './approval-text';
import { useRecallPurchaseRequest } from './api';
import type { RoundKind } from './api';
import styles from './PurchaseRequestApprovalSection.module.css';

const REASON_MAX = 2000;
const reasonSchema = z.object({
  reason: z.string().max(REASON_MAX, 'Keep the reason to 2,000 characters.'),
});

export interface PurchaseRequestRecallDialogProps {
  purchaseRequestId: string;
  name: string;
  kind: RoundKind;
  roundId: string;
  onClose: () => void;
}

/** Recall the open round (032 FR-024): it names the round the person saw, with an optional reason. */
export function PurchaseRequestRecallDialog({
  purchaseRequestId,
  name,
  kind,
  roundId,
  onClose,
}: PurchaseRequestRecallDialogProps) {
  const { show } = useToast();
  const recall = useRecallPurchaseRequest(purchaseRequestId);
  const [problem, setProblem] = useState('');
  const form = useForm({
    resolver: zodResolver(reasonSchema),
    defaultValues: { reason: '' },
    ...CHECK_ON_SAVE,
  });

  async function send(reason: string) {
    try {
      await recall.mutateAsync({ roundId, reason: reason.trim() });
      show({ message: recalledToast(kind, name) });
      onClose();
    } catch (error) {
      const failure = asApiError(error);
      if (!isSessionLost(failure)) {
        setProblem(failure.message);
      }
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={recallTitle(kind, name)}
      confirmLabel={RECALL}
      busy={recall.isPending}
      onConfirm={() => void form.handleSubmit(({ reason }) => send(reason))()}
    >
      <div className={styles.dialogBody}>
        <p>{recallText(kind)}</p>
        {problem && (
          <p role="alert" className={styles.problem}>
            {problem}
          </p>
        )}
        <FormField
          control={form.control}
          name="reason"
          render={(field) => (
            <TextAreaField label={REASON_LABEL} rows={3} maxLength={REASON_MAX} {...field} />
          )}
        />
      </div>
    </ConfirmDialog>
  );
}
