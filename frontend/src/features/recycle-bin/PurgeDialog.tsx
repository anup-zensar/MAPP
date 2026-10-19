import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import type { ApiError } from '@/api/errors';
import { ConfirmDialog, useToast } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { insideEntriesWarning, purgeContentsText, purgeTitle, purgedMessage } from './binText';
import type { BinItem } from './binText';
import { asApiError, closingMessage, refetchBin } from './dialogFailure';
import { usePurge } from './recycleBinApi';
import styles from './dialogs.module.css';

export interface PurgeDialogProps {
  item: BinItem;
  onClose: () => void;
}

/**
 * Asks before deleting an entry permanently (FR-028, FR-030): what goes with it, that it can't be
 * undone, and which other entries can no longer come back. Cancel has the focus.
 */
export function PurgeDialog({ item, onClose }: PurgeDialogProps) {
  const purge = usePurge();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const queryClient = useQueryClient();
  const [failure, setFailure] = useState<ApiError | null>(null);
  const contents = purgeContentsText(item);
  const warning = insideEntriesWarning(item.insideEntryIds?.length ?? 0);

  const confirm = () => {
    setFailure(null);
    purge.mutate(item.id, {
      onSuccess: (result) => {
        onClose();
        show({ message: purgedMessage(result.recordTypeLabel, result.recordName) });
      },
      onError: (error) => {
        const apiError = asApiError(error);
        const message = closingMessage(apiError, item.recordName, dateFormat);
        if (message === undefined) {
          setFailure(apiError);
          return;
        }
        show({ message });
        refetchBin(queryClient);
        onClose();
      },
    });
  };

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={purgeTitle(item)}
      confirmLabel="Delete permanently"
      tone="danger"
      busy={purge.isPending}
      onConfirm={confirm}
    >
      <div className={styles.sentences}>
        {contents && <p>{contents}</p>}
        <p>This can&apos;t be undone.</p>
        {warning && <p>{warning}</p>}
        {failure && (
          <p role="alert" className={styles.failure}>
            {failure.message}
          </p>
        )}
      </div>
    </ConfirmDialog>
  );
}
