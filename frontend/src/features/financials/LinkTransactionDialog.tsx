import { useQuery } from '@tanstack/react-query';
import { CircleAlert } from 'lucide-react';
import { useEffect, useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import { FormDialog, problemOf, TextField, useToast } from '@/components';
import type { FieldProblem } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { fetchLineItemPreview, FINANCIALS_KEY, useLinkTransaction } from './financialsApi';
import type { FinancialTransaction, LineItemPreview } from './financialsApi';
import styles from './LinkTransactionDialog.module.css';

const LINE_ITEM_ID = /^L-[0-9]+$/;
const BAD_ID = 'Line item ID must look like L-1003.';

/** How long the preview waits after the last key. */
export const PREVIEW_DELAY_MS = 300;

export interface LinkTransactionDialogProps {
  transaction: FinancialTransaction;
  onClose: () => void;
}

/**
 * Link to a line item (015 US7, FR-026): type the line item's ID, see its name, activity and plan
 * before saving ("Station signage (L-1007) · Grand opening Shanghai · FY2026 Mobility China"),
 * then Link. A missing line item or one in the recycle bin is refused at the field; a change made
 * meanwhile by someone else is refused as a version conflict.
 */
export function LinkTransactionDialog({ transaction, onClose }: LinkTransactionDialogProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const link = useLinkTransaction();
  const [typed, setTyped] = useState('');
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<string | null>(null);
  const lineItemId = typed.trim().toUpperCase();
  const valid = LINE_ITEM_ID.test(lineItemId);
  // The preview follows the typing a moment later, so every key does not ask the API.
  const previewId = useSettled(valid ? lineItemId : '', PREVIEW_DELAY_MS);
  const preview = useQuery({
    queryKey: [...FINANCIALS_KEY, 'line-item-preview', previewId],
    queryFn: () => fetchLineItemPreview(previewId),
    enabled: previewId !== '',
    retry: false,
  });

  async function submit() {
    if (link.isPending) {
      return;
    }
    if (!valid) {
      setProblems([{ field: 'lineItemId', message: BAD_ID }]);
      return;
    }
    setProblems([]);
    setFailure(null);
    try {
      const linked = await link.mutateAsync({
        id: transaction.id,
        lineItemId,
        version: transaction.version,
      });
      const lineItem = linked.link.path.find((record) => record.type === 'lineItem');
      show({
        message: `'${transaction.name}' linked to '${lineItem ? `${lineItem.name} (${lineItem.id})` : lineItemId}'`,
      });
      onClose();
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind === 'signInRequired' || apiError.kind === 'sessionExpired') {
        onClose();
        return;
      }
      if (apiError.kind === 'validation' && apiError.fields.length > 0) {
        setProblems(apiError.fields);
        return;
      }
      setFailure(conflictMessage(apiError, 'GSAP transaction', dateFormat));
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Link '${transaction.name}' to a line item`}
      description="Enter the ID of the line item GSAP should have named. You can unlink it later."
      submitLabel="Link"
      busy={link.isPending}
      onSubmit={() => void submit()}
    >
      <div className={styles.body}>
        {failure && (
          <p className={styles.failure} role="alert">
            <CircleAlert aria-hidden="true" />
            {failure}
          </p>
        )}
        <TextField
          label="Line item ID"
          name="lineItemId"
          hint="For example L-1007"
          marking="Required"
          value={typed}
          onChange={(text) => {
            setTyped(text);
            setProblems([]);
          }}
          error={problemOf(problems, 'lineItemId')}
          maxLength={20}
        />
        <Preview
          id={previewId}
          stale={previewId !== (valid ? lineItemId : '')}
          loading={preview.isFetching}
          data={preview.data}
          error={preview.error}
        />
      </div>
    </FormDialog>
  );
}

interface PreviewProps {
  id: string;
  stale: boolean;
  loading: boolean;
  data: LineItemPreview | undefined;
  error: unknown;
}

/** "Station signage (L-1007) · Grand opening Shanghai · FY2026 Mobility China", or why not. */
function Preview({ id, stale, loading, data, error }: PreviewProps) {
  let text: string | null = null;
  if (id === '' || stale) {
    text = null;
  } else if (loading) {
    text = `Looking up ${id}…`;
  } else if (data) {
    const plan =
      data.activity.path.find((record) => record.type === 'plan') ?? data.activity.path[0];
    text = [`${data.lineItem.name} (${data.lineItem.id})`, data.activity.name, plan?.name]
      .filter(Boolean)
      .join(' · ');
  } else if (error instanceof ApiError && error.kind === 'notFound') {
    text = `No line item ${id}.`;
  } else if (error) {
    text = "The line item couldn't be looked up. You can still link it.";
  }

  return (
    <p className={styles.preview} aria-live="polite">
      {text && <span className={styles.previewLabel}>Line item</span>}
      {text}
    </p>
  );
}

/** The value once it has stayed the same for `delay` milliseconds. */
function useSettled(value: string, delay: number): string {
  const [settled, setSettled] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setSettled(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return settled;
}
