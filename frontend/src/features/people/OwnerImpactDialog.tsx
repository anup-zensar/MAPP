import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import type { ImpactWarning } from './peopleApi';
import styles from './OwnerImpactDialog.module.css';

export interface OwnerImpactDialogProps {
  /** The warnings of the impact check; the dialog shows while there are any. */
  warnings: ImpactWarning[];
  /** The save is on its way: Save changes shows it and ignores more clicks. */
  busy: boolean;
  /** Back to the form, nothing sent. */
  onKeepEditing: () => void;
  /** Sends the save. */
  onSave: () => void;
}

/**
 * Asks before a person or team form saves a change that leaves plan owners outside their plans'
 * teams (plan "Save flow", FR-013): each warning of the API, then Keep editing (focused) or Save
 * changes.
 */
export function OwnerImpactDialog({
  warnings,
  busy,
  onKeepEditing,
  onSave,
}: OwnerImpactDialogProps) {
  return (
    <ConfirmDialog
      open={warnings.length > 0}
      onOpenChange={(open) => {
        if (!open) {
          onKeepEditing();
        }
      }}
      title="Some plans will need a new owner"
      cancelLabel="Keep editing"
      confirmLabel="Save changes"
      busy={busy}
      onConfirm={onSave}
    >
      <ul className={styles.warnings}>
        {warnings.map((warning) => (
          <li key={warning.message}>{warning.message}</li>
        ))}
      </ul>
    </ConfirmDialog>
  );
}
