import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useBulkPreview, useRunBulk } from './api';
import { Problem } from './BulkDialog';
import styles from './bulk.module.css';
import { countsText, recordsText } from './bulk-text';
import type { BulkResult } from './bulk-types';
import type { BulkActionDialogProps } from './ChangeOwnerDialog';
import { PreviewNotes } from './OutcomeList';

/** One question naming what goes with the records and what is in approval; Cancel has the focus. */
export function DeleteDialog({ records, onClose, onDone }: BulkActionDialogProps) {
  const preview = useBulkPreview({ action: 'delete', recordIds: records.map((r) => r.id) });
  const run = useRunBulk();
  const summary = preview.data?.deleteSummary;
  const count = preview.data?.counts.done ?? records.length;

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={summary ? `Delete ${countsText(summary.selected)}?` : 'Delete'}
      confirmLabel={`Delete ${recordsText(count)}`}
      tone="danger"
      busy={run.isPending || preview.isPending}
      onConfirm={() =>
        run.mutate({ action: 'delete', recordIds: records.map((r) => r.id) }, { onSuccess: onDone })
      }
    >
      {preview.isPending && <Skeleton lines={3} />}
      {preview.error && <Problem error={preview.error} />}
      {preview.data && summary && <Sentences result={preview.data} />}
      {run.error && <Problem error={run.error} />}
    </ConfirmDialog>
  );
}

function Sentences({ result }: { result: BulkResult }) {
  const summary = result.deleteSummary;
  if (!summary) return null;
  const below = countsText(summary.below);
  const lineItems = summary.below.lineItems;
  const goesWith = [
    below,
    lineItems > 0 ? `${lineItems} line ${lineItems === 1 ? 'item' : 'items'}` : '',
  ]
    .filter(Boolean)
    .join(' and ');
  return (
    <div className={styles.sentences}>
      {goesWith && <p>With them go {goesWith}.</p>}
      {summary.submittedOrApproved > 0 && (
        <p>{summary.submittedOrApproved} of them are submitted or approved.</p>
      )}
      <p>Plan totals will update. They move to the recycle bin; you can restore them.</p>
      <PreviewNotes result={result} />
    </div>
  );
}
