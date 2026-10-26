import type { components } from '@/api/generated/schema';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { SETTING_NOUNS } from './settingsNouns';
import type { SettingKind } from './settingsNouns';
import styles from './SettingsTable.module.css';

type ImpactWarning = components['schemas']['ImpactWarning'];

export interface ImpactDialogProps {
  kind: SettingKind;
  /** The record's name as it is saved now. */
  name: string;
  /** What the change would affect, from the record's …/impact check. */
  warnings: ImpactWarning[];
  /** The save is on its way. */
  busy?: boolean;
  onConfirm: () => void;
  /** Keep editing: back to the form, nothing saved. */
  onCancel: () => void;
}

/**
 * Before a change that affects other records is saved (FR-011, FR-012, FR-030): "Save changes to
 * model 'Lubricants 1.0'?" with what will happen, **Save changes** or **Keep editing** (focused).
 */
export function ImpactDialog({
  kind,
  name,
  warnings,
  busy,
  onConfirm,
  onCancel,
}: ImpactDialogProps) {
  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onCancel()}
      title={`Save changes to ${SETTING_NOUNS[kind].noun} '${name}'?`}
      confirmLabel="Save changes"
      cancelLabel="Keep editing"
      busy={busy}
      onConfirm={onConfirm}
    >
      {warnings.length === 1 ? (
        <p>{warnings[0].message}</p>
      ) : (
        <ul className={styles.warnings}>
          {warnings.map((warning) => (
            <li key={warning.message}>{warning.message}</li>
          ))}
        </ul>
      )}
    </ConfirmDialog>
  );
}
