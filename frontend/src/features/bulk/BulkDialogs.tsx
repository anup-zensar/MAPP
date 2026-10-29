import type { BulkKind, BulkLevel, BulkResult, SelectedRecord } from './bulk-types';
import { ChangeFieldDialog } from './ChangeFieldDialog';
import { ChangeOwnerDialog } from './ChangeOwnerDialog';
import { CopyMoveDialog } from './CopyMoveDialog';
import { DeleteDialog } from './DeleteDialog';
import { MixedLevelDialog } from './MixedLevelDialog';
import { SubmitDialog } from './SubmitDialog';

export interface BulkDialogsProps {
  action: BulkKind;
  mixed: boolean;
  records: SelectedRecord[];
  onKeep: (level: BulkLevel) => void;
  onClose: () => void;
  onDone: (result: BulkResult) => void;
}

/** The dialog of the action chosen in the bar, after keeping one level when it needs one. */
export function BulkDialogs({ action, mixed, records, onKeep, onClose, onDone }: BulkDialogsProps) {
  if (mixed) {
    return <MixedLevelDialog action={action} records={records} onKeep={onKeep} onClose={onClose} />;
  }
  const props = { records, onClose, onDone };
  switch (action) {
    case 'changeOwner':
      return <ChangeOwnerDialog {...props} />;
    case 'changeField':
      return <ChangeFieldDialog {...props} />;
    case 'submit':
      return <SubmitDialog {...props} />;
    case 'copy':
    case 'move':
      return <CopyMoveDialog action={action} {...props} />;
    case 'delete':
      return <DeleteDialog {...props} />;
  }
}
