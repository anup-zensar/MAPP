import { useState } from 'react';
import { formatWhole } from '@/format/decimal';
import { plural } from '@/format/text';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { BulkDialog } from './BulkDialog';
import type { BulkActionDialogProps } from './ChangeOwnerDialog';

const MAX_COMMENT = 2000;

/** Submit for approval (FR-012, FR-013): how many go and every reason the others are not ready. */
export function SubmitDialog({ records, onClose, onDone }: BulkActionDialogProps) {
  const [comment, setComment] = useState('');
  const recordIds = records.map((r) => r.id);

  return (
    <BulkDialog
      title="Submit for approval"
      request={{ action: 'submit', recordIds }}
      withChoices={(request) => ({ ...request, comment })}
      submitLabel={(preview) =>
        preview ? `Submit ${plural(preview.counts.done, 'activity', 'activities')}` : 'Submit'
      }
      describe={(preview) => {
        const notReady = preview.counts.skipped + preview.counts.failed;
        return (
          <p>
            {formatWhole(preview.counts.done)} will be submitted · {formatWhole(notReady)}{' '}
            {notReady === 1 ? 'is' : 'are'} not ready
          </p>
        );
      }}
      onClose={onClose}
      onDone={onDone}
    >
      <TextAreaField
        label="Comment for the approvers (optional)"
        name="comment"
        value={comment}
        onChange={setComment}
        maxLength={MAX_COMMENT}
        rows={3}
      />
    </BulkDialog>
  );
}
