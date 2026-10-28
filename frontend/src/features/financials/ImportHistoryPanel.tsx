import { FileText } from 'lucide-react';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useFormatters } from '@/format/useFormatters';
import { useFinancialImports } from './api';
import type { FinancialImport } from './api';
import { importCountsText, NO_ACCESS, NO_DATA, relinkedText } from './financials-text';
import styles from './ImportHistoryPanel.module.css';

const SIMULATED_GSAP_SOURCE = 'Simulated GSAP';

export interface ImportHistoryPanelProps {
  onClose: () => void;
}

/** Import history (015 FR-008; 033 FR-011): every import, newest first, with its counts. */
export function ImportHistoryPanel({ onClose }: ImportHistoryPanelProps) {
  const imports = useFinancialImports();
  const forbidden = imports.error instanceof ApiError && imports.error.kind === 'forbidden';

  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title="Import history"
      description="Every GSAP extract and demo load, newest first."
      footer={<Button onClick={onClose}>Close</Button>}
    >
      {imports.data ? (
        imports.data.items.length === 0 ? (
          <EmptyState icon={<FileText />} title={NO_DATA} />
        ) : (
          // Focusable, so the keyboard can scroll a long history, which holds no link or button.
          // eslint-disable-next-line jsx-a11y-x/no-noninteractive-tabindex -- the list scrolls
          <ol className={styles.list} tabIndex={0} aria-label="Imports, newest first">
            {imports.data.items.map((item) => (
              <ImportItem key={item.id} item={item} />
            ))}
          </ol>
        )
      ) : forbidden ? (
        <p className={styles.muted}>{NO_ACCESS}</p>
      ) : imports.isError ? (
        <ErrorState error={imports.error} onRetry={() => void imports.refetch()} />
      ) : (
        <Skeleton lines={6} height="18px" />
      )}
    </SidePanel>
  );
}

function ImportItem({ item }: { item: FinancialImport }) {
  const { dateTime } = useFormatters();

  return (
    <li className={styles.item}>
      <p className={styles.name}>{item.fileName || item.feedName}</p>
      <p className={styles.meta}>
        {[
          item.source === 'simulatedGsap' ? SIMULATED_GSAP_SOURCE : null,
          item.fileName ? item.feedName : null,
          item.id,
        ]
          .filter(Boolean)
          .join(' · ')}
      </p>
      <p className={styles.meta}>{`${dateTime(item.importedAt)} by ${item.importedBy.name}`}</p>
      <p>{importCountsText(item.counts)}</p>
      {item.counts.relinked > 0 && <p>{relinkedText(item.counts.relinked)}</p>}
    </li>
  );
}
