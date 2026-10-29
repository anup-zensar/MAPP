import { useState } from 'react';
import { SearchField } from '@/ui/SearchField/SearchField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useBulkTargets } from './api';
import { BulkDialog, Problem } from './BulkDialog';
import styles from './bulk.module.css';
import { recordsText } from './bulk-text';
import type { BulkTarget } from './bulk-types';
import type { BulkActionDialogProps } from './ChangeOwnerDialog';

export interface CopyMoveDialogProps extends BulkActionDialogProps {
  action: 'copy' | 'move';
}

/** Copy to… and Move to… (FR-014 – FR-018): a searchable list of the places allowed, then the preview. */
export function CopyMoveDialog({ action, records, onClose, onDone }: CopyMoveDialogProps) {
  const [search, setSearch] = useState('');
  const [target, setTarget] = useState<BulkTarget | null>(null);
  const recordIds = records.map((r) => r.id);
  const targets = useBulkTargets(action, recordIds, search);
  const word = action === 'copy' ? 'Copy' : 'Move';

  return (
    <BulkDialog
      title={`${word} to…`}
      request={target ? { action, recordIds, targetId: target.id } : null}
      submitLabel={(preview) => `${word} ${recordsText(preview?.counts.done ?? records.length)}`}
      describe={(preview) => (
        <p>
          {word} {recordsText(preview.counts.done)} {action === 'copy' ? 'into' : 'to'} &apos;
          {target?.path}
          &apos;.
        </p>
      )}
      onClose={onClose}
      onDone={onDone}
    >
      <SearchField label="Search name, ID or path" value={search} onChange={setSearch} />
      {targets.isPending && <Skeleton lines={3} />}
      {targets.error && <Problem error={targets.error} />}
      {targets.data?.note && <p>{targets.data.note}</p>}
      {targets.data && targets.data.items.length > 0 && (
        <fieldset className={styles.sentences}>
          <legend className="visually-hidden">Where to {action} them</legend>
          <ul className={styles.targets}>
            {targets.data.items.map((t) => (
              <li key={t.id} className={styles.target}>
                <label>
                  <input
                    type="radio"
                    name="target"
                    checked={target?.id === t.id}
                    onChange={() => setTarget(t)}
                  />{' '}
                  {t.path} <span className={styles.id}>{t.id}</span>
                </label>
              </li>
            ))}
          </ul>
        </fieldset>
      )}
    </BulkDialog>
  );
}
