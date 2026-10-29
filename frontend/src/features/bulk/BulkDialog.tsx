import type { ReactNode } from 'react';
import { asApiError } from '@/api/errors';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useBulkPreview, useRunBulk } from './api';
import styles from './bulk.module.css';
import type { BulkRequest, BulkResult } from './bulk-types';
import { PreviewNotes } from './OutcomeList';

export interface BulkDialogProps {
  title: string;
  request: BulkRequest | null;
  children?: ReactNode;
  submitLabel: (preview: BulkResult | undefined) => string;
  describe?: (preview: BulkResult) => ReactNode;
  withChoices?: (request: BulkRequest) => BulkRequest;
  onClose: () => void;
  onDone: (result: BulkResult) => void;
}

/** The choices, the API's preview of them for a request (null: not complete), then the run on confirm. */
export function BulkDialog(props: BulkDialogProps) {
  const { title, request, children, submitLabel, describe, withChoices, onClose, onDone } = props;
  const preview = useBulkPreview(request);
  const run = useRunBulk();
  const nothing = preview.data?.counts.done === 0;

  function confirm() {
    if (nothing) {
      onClose();
    } else if (request && preview.data) {
      run.mutate(withChoices ? withChoices(request) : request, { onSuccess: onDone });
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={title}
      submitLabel={nothing ? 'Close' : submitLabel(preview.data)}
      onSubmit={confirm}
      busy={run.isPending || preview.isFetching}
    >
      {children}
      {request && preview.isPending && <Skeleton lines={2} />}
      {preview.error && <Problem error={preview.error} />}
      {preview.data && (
        <div className={styles.sentences} aria-live="polite">
          {nothing ? (
            <p>Nothing can be {preview.data.verb} — fix the problems first.</p>
          ) : (
            describe?.(preview.data)
          )}
          <PreviewNotes result={preview.data} />
        </div>
      )}
      {run.error && <Problem error={run.error} />}
    </FormDialog>
  );
}

export function Problem({ error }: { error: unknown }) {
  const apiError = asApiError(error);
  return (
    <div role="alert" className={styles.limit}>
      {apiError.fields.length > 0
        ? apiError.fields.map((f) => <p key={f.message}>{f.message}</p>)
        : apiError.message}
    </div>
  );
}
