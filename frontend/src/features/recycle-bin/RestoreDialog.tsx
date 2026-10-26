import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useEffectEvent, useState } from 'react';
import { useNavigate } from 'react-router';
import type { ApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useToast } from '@/ui/Toast/useToast';
import { useFormatters } from '@/format/useFormatters';
import {
  blockedText,
  blockedTitle,
  changeSentence,
  destinationSentence,
  parentEntryLine,
  parentsIntro,
  restoreConfirmLabel,
  restoreTitle,
  restoredHref,
  restoredMessage,
} from './bin-text';
import type { BinItem, RestorePreview } from './bin-text';
import { asApiError, closingMessage, refetchBin } from './dialog-failure';
import { useRestore, useRestorePreview } from './api';
import styles from './dialogs.module.css';

export interface RestoreDialogProps {
  item: BinItem;
  onClose: () => void;
  /** Opens Delete permanently for an entry that can no longer be restored (FR-024). */
  onPurge: (item: BinItem) => void;
}

/**
 * Asks before a restore (FR-018 – FR-026): where the record returns, what changes on the way
 * (status rules, a new name), the notes of its type, and — when its parent is in the recycle bin
 * too — the entries that come back first (**Restore both** / **Restore all <N>**). An entry whose
 * parent was deleted permanently cannot come back: the dialog says why and offers **Delete
 * permanently** instead.
 */
export function RestoreDialog({ item, onClose, onPurge }: RestoreDialogProps) {
  const blockedInList = item.restoreState === 'parentDeletedPermanently';
  const preview = useRestorePreview(item.id, !blockedInList);
  const restore = useRestore();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [failure, setFailure] = useState<ApiError | null>(null);
  const entry = preview.data?.entry ?? item;

  /** Ends the dialog with a message for the failures that end it; true when it did. */
  const endOnFailure = (error: ApiError): boolean => {
    const message = closingMessage(error, item.recordName, dateFormat);
    if (message === undefined) {
      return false;
    }
    show({ message });
    refetchBin(queryClient);
    onClose();
    return true;
  };

  const previewFailed = useEffectEvent((error: Error) => endOnFailure(asApiError(error)));
  useEffect(() => {
    if (preview.error) {
      previewFailed(preview.error);
    }
  }, [preview.error]);

  const close = (open: boolean) => {
    if (!open) {
      onClose();
    }
  };

  if (blockedInList || entry.restoreState === 'parentDeletedPermanently') {
    return (
      <ConfirmDialog
        open
        onOpenChange={close}
        title={blockedTitle(entry)}
        confirmLabel="Delete permanently"
        tone="danger"
        onConfirm={() => {
          onClose();
          onPurge(entry);
        }}
      >
        <p>{blockedText(entry)}</p>
      </ConfirmDialog>
    );
  }

  const parentEntries = preview.data?.parentEntries ?? [];
  const confirm = () => {
    if (!preview.data) {
      // The preview failed: asking again is the way forward.
      void preview.refetch();
      return;
    }
    const destination = preview.data.destination;
    setFailure(null);
    restore.mutate(
      { entryId: item.id, withParents: parentEntries.length > 0 },
      {
        onSuccess: (result) => {
          onClose();
          const href = restoredHref(result.recordType, result.recordId, destination);
          show({
            message: restoredMessage(result.recordTypeLabel, result.recordName),
            action: href
              ? {
                  label: 'View',
                  altText: `Find '${result.recordName}' on the Plans page`,
                  onClick: () => void navigate(href),
                }
              : undefined,
          });
        },
        onError: (error) => {
          const apiError = asApiError(error);
          if (!endOnFailure(apiError)) {
            setFailure(apiError);
          }
        },
      },
    );
  };

  return (
    <ConfirmDialog
      open
      onOpenChange={close}
      title={restoreTitle(entry)}
      confirmLabel={restoreConfirmLabel(entry, parentEntries.length)}
      busy={preview.isFetching || restore.isPending}
      onConfirm={confirm}
    >
      {preview.data ? (
        <PreviewText preview={preview.data} />
      ) : preview.error ? (
        <p role="alert">{asApiError(preview.error).message}</p>
      ) : (
        <Skeleton lines={3} />
      )}
      {failure && (
        <p role="alert" className={styles.failure}>
          {failure.message}
        </p>
      )}
    </ConfirmDialog>
  );
}

/** The parents that come back first, where it returns, what changes and the notes. */
function PreviewText({ preview }: { preview: RestorePreview }) {
  const { entry, parentEntries, changes, notes } = preview;
  return (
    <div className={styles.sentences}>
      {parentEntries.length > 0 && entry.missingParent && (
        <div>
          <p>{parentsIntro(entry.missingParent)}</p>
          <ul className={styles.list}>
            {parentEntries.map((parent) => (
              <li key={parent.id}>{parentEntryLine(parent)}</li>
            ))}
          </ul>
        </div>
      )}
      <p>{destinationSentence(preview)}</p>
      {changes.map((change) => (
        <p key={`${change.recordId}-${change.field}`}>{changeSentence(change, entry)}</p>
      ))}
      {notes.map((note) => (
        <p key={note}>{note}</p>
      ))}
    </div>
  );
}
