import { useState } from 'react';
import type { ApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { emptiedMessage, emptyBinText } from './bin-text';
import type { BinItem } from './bin-text';
import { asApiError } from './dialog-failure';
import { useEmptyBin } from './api';
import styles from './dialogs.module.css';

export interface EmptyBinDialogProps {
  /** Every entry the page lists — not only those the search shows. */
  items: BinItem[];
  onClose: () => void;
}

/**
 * Asks before emptying the recycle bin (FR-029, FR-030): how many entries and what is inside them.
 * Only the entries the person saw here are deleted, even when someone added one meanwhile.
 */
export function EmptyBinDialog({ items, onClose }: EmptyBinDialogProps) {
  const emptyBin = useEmptyBin();
  const { show } = useToast();
  const [failure, setFailure] = useState<ApiError | null>(null);

  const confirm = () => {
    setFailure(null);
    emptyBin.mutate(
      items.map((item) => item.id),
      {
        onSuccess: (result) => {
          onClose();
          show({ message: emptiedMessage(result.count) });
        },
        onError: (error) => setFailure(asApiError(error)),
      },
    );
  };

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title="Empty the recycle bin?"
      confirmLabel="Empty recycle bin"
      tone="danger"
      busy={emptyBin.isPending}
      onConfirm={confirm}
    >
      <div className={styles.sentences}>
        <p>{emptyBinText(items)}</p>
        <p>This can&apos;t be undone.</p>
        {failure && (
          <p role="alert" className={styles.failure}>
            {failure.message}
          </p>
        )}
      </div>
    </ConfirmDialog>
  );
}
