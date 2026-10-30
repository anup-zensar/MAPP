import { Download } from 'lucide-react';
import { useId, useState } from 'react';
import type { ChangeEvent } from 'react';
import { useNavigate, useParams } from 'react-router';
import { ApiError, asApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useToast } from '@/ui/Toast/useToast';
import { usePageTitle } from '@/ui/usePageTitle';
import {
  downloadExport,
  downloadTemplate,
  useCheckImport,
  useRunImport,
  useWorkbookPlan,
} from './api';
import type { ImportCheck, ImportUpload } from './api';
import { CheckResult } from './CheckResult';
import { base64Of, MAX_FILE_BYTES } from './files';
import { ImportsList } from './ImportsList';
import { LoadFailed } from './LoadFailed';
import styles from './workbook.module.css';
import { CHOOSE_FILE, FILE_TOO_LARGE } from './workbook-text';

const STEPS = ['1 Get the template', '2 Upload', '3 Check and import'];

/** Import from Excel: template or export, upload, check every row, then import the ready rows. */
export function ImportPage() {
  usePageTitle('Import from Excel');
  const planId = useParams().planId ?? '';
  const plan = useWorkbookPlan(planId);
  const navigate = useNavigate();
  const { show } = useToast();
  const check = useCheckImport(planId);
  const run = useRunImport(planId);
  const inputId = useId();
  const [upload, setUpload] = useState<ImportUpload | null>(null);
  const [result, setResult] = useState<ImportCheck | null>(null);
  const [fileProblem, setFileProblem] = useState<string | null>(null);

  async function choose(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    setUpload(null);
    setResult(null);
    setFileProblem(null);
    if (!file) return;
    if (!/\.(xlsx|csv)$/i.test(file.name)) {
      setFileProblem(CHOOSE_FILE);
    } else if (file.size > MAX_FILE_BYTES) {
      setFileProblem(FILE_TOO_LARGE);
    } else {
      setUpload({ fileName: file.name, content: await base64Of(file) });
    }
  }

  function runCheck() {
    if (!upload) {
      setFileProblem(CHOOSE_FILE);
      return;
    }
    check.mutate(upload, {
      onSuccess: setResult,
      onError: (error) => setFileProblem(messageOf(error)),
    });
  }

  function runImport() {
    if (!upload) return;
    run.mutate(upload, {
      onSuccess: (detail) => void navigate(`/imports/${encodeURIComponent(detail.import.id)}`),
      onError: (error) => show({ message: messageOf(error) }),
    });
  }

  async function download(get: (planId: string) => Promise<string>) {
    try {
      await get(planId);
    } catch (error) {
      show({ message: messageOf(error) });
    }
  }

  if (plan.isPending) return <Skeleton lines={6} />;
  if (plan.isError) return <LoadFailed error={plan.error} onRetry={() => void plan.refetch()} />;
  const step = result ? 2 : upload ? 1 : 0;

  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Plans', to: '/plans' },
          { label: plan.data.name, to: `/plans/${encodeURIComponent(planId)}` },
          { label: 'Import from Excel' },
        ]}
        eyebrow="Import"
        title="Import from Excel"
        description={`Build or update '${plan.data.name}' from a spreadsheet: campaigns, initiatives, activities and line items.`}
      />
      <div className={styles.stack}>
        <ol className={styles.steps} aria-label="Steps">
          {STEPS.map((label, i) => (
            <li key={label} aria-current={i === step ? 'step' : undefined}>
              {label}
            </li>
          ))}
        </ol>
        <Card title="Get the template">
          <p className={styles.text}>
            The template lists this plan&apos;s campaigns, initiatives and every allowed value. The
            export holds the plan&apos;s records with their IDs: change values, add rows without an
            ID, then upload it.
          </p>
          <div className={styles.row}>
            <Button icon={<Download />} onClick={() => void download(downloadTemplate)}>
              Download template
            </Button>
            <Button icon={<Download />} onClick={() => void download(downloadExport)}>
              Export to Excel
            </Button>
          </div>
        </Card>
        <Card title="Upload">
          <label htmlFor={inputId} className={styles.label}>
            Choose a file
          </label>
          <p id={`${inputId}-hint`} className={styles.note}>
            The plan workbook (.xlsx) or an Activities sheet saved as .csv, up to 5 MB and 5,000
            rows.
          </p>
          <div className={styles.row}>
            <input
              id={inputId}
              type="file"
              accept=".xlsx,.csv"
              aria-describedby={`${inputId}-hint`}
              aria-invalid={fileProblem ? true : undefined}
              onChange={(event) => void choose(event)}
            />
            <Button
              variant={result ? 'secondary' : 'primary'}
              aria-busy={check.isPending || undefined}
              onClick={runCheck}
            >
              {check.isPending ? 'Checking your file…' : 'Check file'}
            </Button>
          </div>
          {fileProblem && (
            <p role="alert" className={styles.problem}>
              {fileProblem}
            </p>
          )}
        </Card>
        {result && (
          <Card title="Check and import">
            <CheckResult check={result} />
            <div className={styles.row}>
              {result.ready > 0 && !result.nothingToChange ? (
                <Button
                  variant="primary"
                  aria-busy={run.isPending || undefined}
                  onClick={runImport}
                >
                  Import {result.ready} rows
                </Button>
              ) : (
                <p className={styles.note}>
                  {result.nothingToChange
                    ? 'The file holds what the plan holds — nothing to import.'
                    : 'No rows are ready to import — fix the file and upload it again.'}
                </p>
              )}
            </div>
          </Card>
        )}
        <ImportsList planId={planId} />
      </div>
    </>
  );
}

function messageOf(error: unknown): string {
  const apiError = error instanceof ApiError ? error : asApiError(error);
  return apiError.fields.map((f) => f.message).join(' ') || apiError.message;
}
