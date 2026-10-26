import { ConfirmDialog } from '../ConfirmDialog/ConfirmDialog';
import type { ClearedValue } from './rules';
import styles from './ShellAttributeFields.module.css';

export interface ClearValuesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** What saving would clear: valuesToClear(form, values, previousValues). */
  cleared: ClearedValue[];
  /** Saves; the API removes the values listed. */
  onConfirm: () => void;
  busy?: boolean;
}

/**
 * Asks before a save clears dependent values that no longer fit their parent (feature 002,
 * FR-025, design system §7): "Region: RETAIL – CHINA". Keep editing has the focus.
 */
export function ClearValuesDialog({
  open,
  onOpenChange,
  cleared,
  onConfirm,
  busy,
}: ClearValuesDialogProps) {
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Clear values that no longer fit?"
      confirmLabel="Clear and save"
      cancelLabel="Keep editing"
      onConfirm={onConfirm}
      busy={busy}
    >
      <ul className={styles.dialogList}>
        {cleared.map((item) => (
          <li key={item.attributeId}>
            {item.label}: {item.optionLabels.join(', ')}
          </li>
        ))}
      </ul>
    </ConfirmDialog>
  );
}
