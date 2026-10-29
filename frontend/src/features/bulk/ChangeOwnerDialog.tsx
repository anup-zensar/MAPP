import { useState } from 'react';
import { PersonField } from '@/features/people';
import { BulkDialog } from './BulkDialog';
import { recordsText } from './bulk-text';
import type { BulkResult, SelectedRecord } from './bulk-types';

export interface BulkActionDialogProps {
  records: SelectedRecord[];
  onClose: () => void;
  onDone: (result: BulkResult) => void;
}

/** Change owner (FR-008): one active person becomes the owner of every selected record. */
export function ChangeOwnerDialog({ records, onClose, onDone }: BulkActionDialogProps) {
  const [ownerId, setOwnerId] = useState('');
  const request = ownerId
    ? { action: 'changeOwner' as const, recordIds: records.map((r) => r.id), ownerId }
    : null;

  return (
    <BulkDialog
      title="Change owner"
      request={request}
      submitLabel={() => 'Change owner'}
      describe={(preview) => <p>Set the new owner on {recordsText(preview.counts.done)}.</p>}
      onClose={onClose}
      onDone={onDone}
    >
      <PersonField label="New owner" name="ownerId" value={ownerId} onChange={setOwnerId} />
    </BulkDialog>
  );
}
