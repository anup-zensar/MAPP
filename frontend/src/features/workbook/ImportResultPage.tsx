import { useState } from 'react';
import { Link, useParams } from 'react-router';
import { asApiError } from '@/api/errors';
import { useFormatters } from '@/format/useFormatters';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { recordHref } from '@/ui/record-links';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useToast } from '@/ui/Toast/useToast';
import { usePageTitle } from '@/ui/usePageTitle';
import { useImportDetail, useUndoImport } from './api';
import type { ImportDetail } from './api';
import { ProblemsTable } from './CheckResult';
import styles from './workbook.module.css';
import { LEVEL_PLURALS, totalsText } from './workbook-text';

/** The results of one import, its records and Undo import (FR-035, FR-036). */
export function ImportResultPage() {
  usePageTitle('Import results');
  const importId = useParams().importId ?? '';
  const detail = useImportDetail(importId);
  const undo = useUndoImport(importId);
  const format = useFormatters();
  const { show } = useToast();
  const [confirming, setConfirming] = useState(false);

  if (detail.isPending) return <Skeleton lines={6} />;
  if (detail.isError)
    return <ErrorState error={detail.error} onRetry={() => void detail.refetch()} />;
  const { import: imp } = detail.data;

  function confirmUndo() {
    undo.mutate(undefined, {
      onSuccess: () => show({ message: 'Import undone — the records are in the recycle bin' }),
      onError: (error) => show({ message: asApiError(error).message }),
      onSettled: () => setConfirming(false),
    });
  }

  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Plans', to: '/plans' },
          { label: 'Import from Excel', to: `/plans/${encodeURIComponent(imp.planId)}/import` },
          { label: imp.fileName },
        ]}
        eyebrow="Import results"
        title={imp.fileName}
        description={`Imported by ${imp.importedBy.name} on ${format.dateTime(imp.importedAt)}.`}
        actions={
          detail.data.undo && (
            <Button variant="danger" onClick={() => setConfirming(true)}>
              Undo import
            </Button>
          )
        }
      />
      <div className={styles.stack}>
        <Card title="Summary">
          <p className={styles.text}>Created {totalsText(imp.created)}.</p>
          <p className={styles.text}>Updated {totalsText(imp.updated)}.</p>
          <p className={styles.text}>
            {imp.rowsLeftOut} of {imp.rowsRead} rows were left out.
          </p>
          {imp.undoneAt && imp.undoneBy && (
            <p className={styles.text}>
              Undone by {imp.undoneBy.name} on {format.dateTime(imp.undoneAt)}.
            </p>
          )}
        </Card>
        <Records detail={detail.data} />
        {detail.data.problems.length > 0 && (
          <Card title="Rows left out">
            <ProblemsTable problems={detail.data.problems} fileName={imp.fileName} />
          </Card>
        )}
      </div>
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={`Undo import of '${imp.fileName}'?`}
        confirmLabel="Undo import"
        tone="danger"
        busy={undo.isPending}
        onConfirm={confirmUndo}
      >
        <p>
          The{' '}
          {totalsText(
            detail.data.undo ?? { campaigns: 0, initiatives: 0, activities: 0, lineItems: 0 },
          )}{' '}
          it created move to the recycle bin, with anything added under them since. Its updates stay
          as they are.
        </p>
        {(detail.data.undo?.alreadyDeleted ?? 0) > 0 && (
          <p>{detail.data.undo?.alreadyDeleted} were already deleted.</p>
        )}
      </ConfirmDialog>
    </>
  );
}

function Records({ detail }: { detail: ImportDetail }) {
  if (detail.records.length === 0 && detail.kept.length === 0) return null;
  return (
    <Card title="Records">
      <ul className={styles.list}>
        {detail.records.map((r) => (
          <li key={r.id}>
            {r.exists && r.level !== 'campaign' && r.level !== 'initiative' ? (
              <Link to={recordHref(r.level, r.id)}>{r.name}</Link>
            ) : (
              r.name
            )}{' '}
            · {r.id} · {LEVEL_PLURALS[r.level]} · {r.action}
            {!r.exists && ' · no longer exists'}
          </li>
        ))}
      </ul>
      {detail.kept.length > 0 && (
        <>
          <h3 className={styles.label}>Kept when the import was undone</h3>
          <ul className={styles.list}>
            {detail.kept.map((k) => (
              <li key={k.id}>
                {k.name} ({k.id}): {k.reason}
              </li>
            ))}
          </ul>
        </>
      )}
    </Card>
  );
}
