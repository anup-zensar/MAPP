import { ConfirmDialog } from '@/components';
import type { ReapprovalChange } from './activityFormState';
import styles from './ReapprovalDialog.module.css';

export interface ReapprovalDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The activity's name. */
  name: string;
  /** The changes that restart approval, old → new. */
  changes: ReapprovalChange[];
  /**
   * What the typed dates do to the line items' spreads — feature 007's sentence, the one under the
   * dates (useSpreadEffectText, D-150) — or null when the dates do not change them.
   */
  spreadEffect: string | null;
  /** Saves with restartApproval. */
  onConfirm: () => void;
  busy?: boolean;
}

/**
 * Asks before a change of planned cost or dates sends a submitted or approved activity back to
 * Created (FR-027): the changes, old → new, and what happens to the line items' spreads when the
 * dates change. **Keep editing** has the focus.
 */
export function ReapprovalDialog({
  open,
  onOpenChange,
  name,
  changes,
  spreadEffect,
  onConfirm,
  busy,
}: ReapprovalDialogProps) {
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="This activity will need approval again"
      confirmLabel="Save and restart approval"
      cancelLabel="Keep editing"
      onConfirm={onConfirm}
      busy={busy}
    >
      <p>Saving returns &apos;{name}&apos; to Created; it must be submitted for approval again.</p>
      <ul className={styles.changes}>
        {changes.map((change) => (
          <li key={change.label}>
            {change.label}: {change.old} → {change.new}
          </li>
        ))}
      </ul>
      {spreadEffect && <p>{spreadEffect}</p>}
    </ConfirmDialog>
  );
}
