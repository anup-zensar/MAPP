import { FileText } from 'lucide-react';
import { ApiError } from '@/api/errors';
import { Button, EmptyState, ErrorState, SidePanel, Skeleton } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { useFinancialImports } from './financialsApi';
import type { FinancialImport } from './financialsApi';
import { importCountsText, NO_ACCESS, NO_DATA, relinkedText } from './financialsText';
import styles from './ImportHistoryPanel.module.css';

export interface ImportHistoryPanelProps {
  onClose: () => void;
}

/**
 * Import history (015 FR-008): every import, newest first, with its file name, source, who made
 * it, when, and its counts.
 */
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
          <ol className={styles.list}>
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
        {[item.fileName ? item.feedName : null, item.id].filter(Boolean).join(' · ')}
      </p>
      <p className={styles.meta}>{`${dateTime(item.importedAt)} by ${item.importedBy.name}`}</p>
      <p>{importCountsText(item.counts)}</p>
      {item.counts.relinked > 0 && <p>{relinkedText(item.counts.relinked)}</p>}
    </li>
  );
}
