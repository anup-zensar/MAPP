import { ApiError, conflictMessage } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { useFormatters } from '@/lib/useFormatters';
import { useUpdateFieldHelp } from './fieldHelpApi';
import type { FieldHelpEntry } from './fieldHelpApi';

export interface ResetHelpDialogProps {
  entry: FieldHelpEntry;
  /** The version the reset is made from (the panel's, after a conflict). */
  version?: number;
  onDone: () => void;
  onCancel: () => void;
}

/**
 * "Reset the help of 'Planned cost'?" (016 FR-046, US7-3): **Reset to default** brings the
 * default text back (the chip then reads "Default"); Cancel has the focus. A failure, such as a
 * newer change by someone else, is said in a message.
 */
export function ResetHelpDialog({ entry, version, onDone, onCancel }: ResetHelpDialogProps) {
  const update = useUpdateFieldHelp();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const name = `'${entry.label}'`;

  async function reset() {
    try {
      await update.mutateAsync({
        key: entry.key,
        body: { text: null, version: version ?? entry.version },
      });
      show({ message: `Help of ${name} reset to the default` });
      onDone();
    } catch (error) {
      const failure = error instanceof ApiError ? error : new ApiError('other');
      if (failure.kind !== 'signInRequired' && failure.kind !== 'sessionExpired') {
        show({ message: conflictMessage(failure, 'help text', dateFormat) });
      }
      onCancel();
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onCancel()}
      title={`Reset the help of ${name}?`}
      confirmLabel="Reset to default"
      busy={update.isPending}
      onConfirm={() => void reset()}
    >
      <p>The default text written by MAPP comes back. The edited text stays in the activity log.</p>
    </ConfirmDialog>
  );
}
