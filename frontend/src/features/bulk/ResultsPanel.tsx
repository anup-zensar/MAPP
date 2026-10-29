import { Link } from 'react-router';
import { Button } from '@/ui/Button/Button';
import { recordHref } from '@/ui/record-links';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import styles from './bulk.module.css';
import { resultSummary } from './bulk-text';
import type { BulkResult, SelectedRecord } from './bulk-types';
import { OutcomeList } from './OutcomeList';

export interface ResultsPanelProps {
  result: BulkResult;
  onSelectFailed: (records: SelectedRecord[]) => void;
  onClose: () => void;
}

/** The summary of a bulk action with every skipped and failed record and Select failed items (FR-021). */
export function ResultsPanel({ result, onSelectFailed, onClose }: ResultsPanelProps) {
  const notDone = result.outcomes.filter((o) => o.outcome !== 'done');
  const copies = result.outcomes.filter((o) => o.newRecordId);

  return (
    <SidePanel
      open
      onOpenChange={(open) => !open && onClose()}
      title="Results"
      description={resultSummary(result)}
      footer={
        notDone.length > 0 ? (
          <Button
            variant="primary"
            onClick={() =>
              onSelectFailed(
                notDone.map((o) => ({
                  id: o.recordId,
                  level: o.level,
                  name: o.name,
                  planId: o.planId,
                })),
              )
            }
          >
            Select failed items
          </Button>
        ) : (
          <Button onClick={onClose}>Close</Button>
        )
      }
    >
      {copies.length > 0 && (
        <ul className={styles.list} aria-label="New records">
          {copies.map((o) => (
            <li key={o.recordId}>
              {o.name}: new{' '}
              <Link to={recordHref(o.level, o.newRecordId ?? '', o.planId)}>{o.newRecordId}</Link>
            </li>
          ))}
        </ul>
      )}
      <OutcomeList result={result} outcome="skipped" title="Skipped" />
      <OutcomeList result={result} outcome="failed" title="Failed" />
    </SidePanel>
  );
}
