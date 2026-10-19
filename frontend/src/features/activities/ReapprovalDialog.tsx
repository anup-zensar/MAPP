import { ConfirmDialog } from '@/components';
import { plural } from '@/lib/text';
import type { ReapprovalChange } from './activityFormState';
import { lineItemsText } from './activityText';
import styles from './ReapprovalDialog.module.css';

export interface ReapprovalDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The activity's name. */
  name: string;
  /** The changes that restart approval, old → new. */
  changes: ReapprovalChange[];
  /** The activity's line items: re-spread (automatic spreads) and kept (custom spreads). */
  lineItems: { count: number; autoSpread: number; customSpread: number };
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
  lineItems,
  onConfirm,
  busy,
}: ReapprovalDialogProps) {
  const datesChange = changes.some((change) => change.label !== 'Planned cost');
  const { autoSpread, customSpread } = lineItems;

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
      {datesChange && lineItems.count > 0 && (
        <p>
          {lineItemsText(autoSpread)} will be re-spread over the new dates;{' '}
          {plural(customSpread, 'custom spread')} {customSpread === 1 ? 'is' : 'are'} kept as typed.
        </p>
      )}
    </ConfirmDialog>
  );
}
