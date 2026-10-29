import { useState } from 'react';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import styles from './bulk.module.css';
import { actionWord, levelText, levelsPresent } from './bulk-text';
import type { BulkKind, BulkLevel, SelectedRecord } from './bulk-types';

export interface MixedLevelDialogProps {
  action: BulkKind;
  records: SelectedRecord[];
  onKeep: (level: BulkLevel) => void;
  onClose: () => void;
}

const KEEP_WORDS: Record<BulkLevel, string> = {
  campaign: 'Keep only campaigns',
  initiative: 'Keep only initiatives',
  activity: 'Keep only activities',
};

/** Change field, Copy to… and Move to… work on one level: the person keeps one and goes on (FR-007). */
export function MixedLevelDialog({ action, records, onKeep, onClose }: MixedLevelDialogProps) {
  const present = levelsPresent(records);
  const [level, setLevel] = useState<BulkLevel>(present[0] ?? 'activity');

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`${actionWord(action)} works on one level at a time`}
      submitLabel="Continue"
      onSubmit={() => onKeep(level)}
    >
      <fieldset className={styles.sentences}>
        <legend>Which records should it work on?</legend>
        {present.map((l) => (
          <label key={l}>
            <input type="radio" name="level" checked={level === l} onChange={() => setLevel(l)} />{' '}
            {KEEP_WORDS[l]} ({levelText(l, records.filter((r) => r.level === l).length)})
          </label>
        ))}
      </fieldset>
    </FormDialog>
  );
}
