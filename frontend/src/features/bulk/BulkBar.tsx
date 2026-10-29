import { useState } from 'react';
import { asApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { useToast } from '@/ui/Toast/useToast';
import { useUndoBulk } from './api';
import styles from './bulk.module.css';
import {
  LIMIT_MESSAGE,
  MAX_RECORDS,
  levelsPresent,
  recordsText,
  resultSummary,
  selectionSummary,
} from './bulk-text';
import type { BulkKind, BulkResult, SelectedRecord } from './bulk-types';
import { BulkDialogs } from './BulkDialogs';
import { ResultsPanel } from './ResultsPanel';
import type { Selection } from './useSelection';

export interface BulkBarProps {
  selection: Selection;
  /** Every selectable record matching the search and filters, for Select all N matching (FR-002). */
  matching: SelectedRecord[];
}

const ACTIONS: { action: BulkKind; label: string }[] = [
  { action: 'changeOwner', label: 'Change owner' },
  { action: 'changeField', label: 'Change field' },
  { action: 'submit', label: 'Submit for approval' },
  { action: 'copy', label: 'Copy to…' },
  { action: 'move', label: 'Move to…' },
];
const ONE_LEVEL: BulkKind[] = ['changeField', 'copy', 'move'];

/** The bar under a list while anything is selected: the count by level and every action (FR-004 – FR-007). */
export function BulkBar({ selection, matching }: BulkBarProps) {
  const [action, setAction] = useState<BulkKind | null>(null);
  const [mixed, setMixed] = useState(false);
  const [result, setResult] = useState<BulkResult | null>(null);
  const { show } = useToast();
  const undo = useUndoBulk();
  const records = selection.records;

  function start(next: BulkKind) {
    if (records.length > MAX_RECORDS) {
      show({ message: LIMIT_MESSAGE });
      return;
    }
    setMixed(ONE_LEVEL.includes(next) && levelsPresent(records).length > 1);
    setAction(next);
  }

  function done(finished: BulkResult) {
    setAction(null);
    selection.clear();
    const problems = finished.counts.skipped + finished.counts.failed > 0;
    if (finished.action === 'delete' && finished.id) {
      showUndo(finished.id, finished.counts.done);
    } else {
      show({
        message: resultSummary(finished),
        action: {
          label: 'View details',
          altText: 'The results panel lists every record',
          onClick: () => setResult(finished),
        },
      });
    }
    if (problems) setResult(finished);
  }

  function showUndo(bulkActionId: string, count: number) {
    show({
      message: `${recordsText(count)} moved to the recycle bin`,
      action: {
        label: 'Undo',
        altText: 'Restore them from the recycle bin',
        onClick: () =>
          undo.mutate(bulkActionId, {
            onSuccess: (undone) =>
              show({ message: `${recordsText(undone.restored.length)} restored` }),
            onError: (error) => show({ message: asApiError(error).message }),
          }),
      },
    });
  }

  return (
    <>
      {records.length > 0 && (
        <div role="region" aria-label="Selection" className={styles.bar}>
          <span className={styles.summary}>{selectionSummary(records)}</span>
          {matching.length > records.length && (
            <button
              type="button"
              className={styles.linkButton}
              onClick={() => selection.select(matching)}
            >
              Select all {matching.length} matching
            </button>
          )}
          <div className={styles.actions}>
            {ACTIONS.map((a) => (
              <Button key={a.action} size="sm" onClick={() => start(a.action)}>
                {a.label}
              </Button>
            ))}
            <Button size="sm" variant="danger" onClick={() => start('delete')}>
              Delete
            </Button>
            <Button size="sm" variant="tertiary" onClick={selection.clear}>
              Clear
            </Button>
          </div>
          {records.length > MAX_RECORDS && <p className={styles.limit}>{LIMIT_MESSAGE}</p>}
        </div>
      )}
      {action && (
        <BulkDialogs
          action={action}
          mixed={mixed}
          records={records}
          onKeep={(level) => {
            selection.replace(records.filter((r) => r.level === level));
            setMixed(false);
          }}
          onClose={() => setAction(null)}
          onDone={done}
        />
      )}
      {result && (
        <ResultsPanel
          result={result}
          onSelectFailed={(failed) => {
            selection.replace(failed);
            setResult(null);
          }}
          onClose={() => setResult(null)}
        />
      )}
    </>
  );
}
