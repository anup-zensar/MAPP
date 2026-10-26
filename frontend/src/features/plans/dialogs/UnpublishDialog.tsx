import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormField } from '@/ui/FormField/FormField';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { unpublishSchema } from '../schemas';
import type { UnpublishValues } from '../schemas';

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
  const form = useForm<UnpublishValues>({
    resolver: zodResolver(unpublishSchema),
    defaultValues: { reason: '' },
    ...CHECK_ON_SAVE,
  });
  const [busy, setBusy] = useState(false);

  const submit = form.handleSubmit(async ({ reason }) => {
    setBusy(true);
    try {
      const problem = await onUnpublish(reason);
      if (problem) {
        form.setError('reason', { message: problem });
      }
    } finally {
      setBusy(false);
    }
  });

  return (
    <FormDialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={`Unpublish plan '${planName}'?`}
      submitLabel="Unpublish plan"
      onSubmit={() => void submit()}
      busy={busy}
    >
      <FormField
        control={form.control}
        name="reason"
        render={(field) => (
          <TextAreaField
            label="Why are you unpublishing this plan?"
            marking="Required"
            hint="The plan returns to Draft. Your reason is recorded in the activity log."
            maxLength={500}
            rows={3}
            {...field}
          />
        )}
      />
    </FormDialog>
  );
}
