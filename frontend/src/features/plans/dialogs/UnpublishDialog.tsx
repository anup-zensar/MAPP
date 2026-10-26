import { useState } from 'react';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';

export interface UnpublishDialogProps {
  open: boolean;
  planName: string;
  /**
   * Unpublishes with the reason. Answers the problem with the reason to show under the field, or
   * null when the plan was unpublished (the caller then closes the dialog).
   */
  onUnpublish: (reason: string) => Promise<string | null>;
  onClose: () => void;
}

/**
 * Unpublish a plan (FR-025): asks why (1 to 500 characters, checked by the API) and returns the
 * plan to Draft; the reason goes into the activity log. Give it a new `key` each time it opens, so
 * it starts empty.
 */
export function UnpublishDialog({ open, planName, onUnpublish, onClose }: UnpublishDialogProps) {
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    try {
      setError((await onUnpublish(reason)) ?? undefined);
    } finally {
      setBusy(false);
    }
  }

  return (
    <FormDialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={`Unpublish plan '${planName}'?`}
      submitLabel="Unpublish plan"
      onSubmit={() => void submit()}
      busy={busy}
    >
      <TextAreaField
        label="Why are you unpublishing this plan?"
        name="reason"
        marking="Required"
        hint="The plan returns to Draft. Your reason is recorded in the activity log."
        maxLength={500}
        rows={3}
        value={reason}
        onChange={setReason}
        error={error}
      />
    </FormDialog>
  );
}
