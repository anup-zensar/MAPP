import { Link } from 'react-router';
import { useFormatters } from '@/format/useFormatters';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { usePlanImports } from './api';
import styles from './workbook.module.css';
import { totalsText } from './workbook-text';

/** The plan's imports, newest first, each with its status and a link to its results (FR-035). */
export function ImportsList({ planId }: { planId: string }) {
  const imports = usePlanImports(planId);
  const format = useFormatters();

  return (
    <Card title="Earlier imports">
      {imports.isPending && <Skeleton lines={2} />}
      {imports.isError && (
        <ErrorState error={imports.error} onRetry={() => void imports.refetch()} />
      )}
      {imports.data?.items.length === 0 && (
        <p className={styles.note}>No imports yet — import from a spreadsheet.</p>
      )}
      {imports.data && imports.data.items.length > 0 && (
        <ul className={styles.list}>
          {imports.data.items.map((imp) => (
            <li key={imp.id} className={styles.row}>
              <Link to={`/imports/${encodeURIComponent(imp.id)}`}>{imp.fileName}</Link>
              <span>
                {format.dateTime(imp.importedAt)} · {imp.importedBy.name} · created{' '}
                {totalsText(imp.created)} · updated {totalsText(imp.updated)}
              </span>
              <StatusChip tone={imp.status === 'undone' ? 'neutral' : 'positive'}>
                {imp.status === 'undone' ? 'Undone' : 'Imported'}
              </StatusChip>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
