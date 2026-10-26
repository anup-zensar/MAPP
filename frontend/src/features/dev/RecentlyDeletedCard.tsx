import { RotateCcw } from 'lucide-react';
import { ApiError, resolvedMessage } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { useToast } from '@/ui/Toast/useToast';
import { useFormatters } from '@/format/useFormatters';
import { useRestoreFromBin } from './api';
import type { RecycleBinEntry } from './api';
import { deletedText, recordTitle } from './sample-text';
import styles from './SampleRecordsPage.module.css';
import { useFailureMessage } from './useFailureMessage';

/** What was deleted and is not restored yet, newest first, each with Restore (FR-025). */
export function RecentlyDeletedCard({ entries }: { entries: RecycleBinEntry[] }) {
  const format = useFormatters();
  const { show } = useToast();
  const showFailure = useFailureMessage();
  const restoreFromBin = useRestoreFromBin();

  async function restore(entry: RecycleBinEntry) {
    // One restore at a time: a second click while one is on its way is ignored.
    if (restoreFromBin.isPending) {
      return;
    }
    try {
      const { details } = await restoreFromBin.mutateAsync(entry.id);
      // The details say what the restore changed, such as a new name (FR-025).
      const restored = `${recordTitle(entry)} restored`;
      show({ message: details ? `${restored}. ${details}` : restored });
    } catch (error) {
      if (error instanceof ApiError && error.kind === 'alreadyResolved') {
        show({ message: resolvedMessage(error, entry.recordName, format.dateFormat) });
      } else {
        showFailure(error, () => restore(entry));
      }
    }
  }

  return (
    <Card title="Recently deleted">
      <ul role="list" className={styles.list}>
        {entries.map((entry) => (
          <li key={entry.id} className={styles.binEntry}>
            <p>{deletedText(entry, format.dateTime)}</p>
            <Button
              size="sm"
              icon={<RotateCcw />}
              aria-label={`Restore ${entry.recordTypeLabel} ${entry.recordName}`}
              onClick={() => restore(entry)}
            >
              Restore
            </Button>
          </li>
        ))}
      </ul>
    </Card>
  );
}
