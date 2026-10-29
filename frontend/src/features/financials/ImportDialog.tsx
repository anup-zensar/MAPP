import { zodResolver } from '@hookform/resolvers/zod';
import { CircleAlert, Download, LoaderCircle } from 'lucide-react';
import { useId, useRef, useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { downloadSampleFile, useCheckImport, useImportExtract } from './api';
import type { FinancialImportCheck, FinancialImportRequest } from './api';
import { importButtonLabel, importedToast, refusedText, relinkedText } from './financials-text';
import { ImportCheckResult } from './ImportCheckResult';
import { importFileSchema } from './schemas';
import styles from './ImportDialog.module.css';

const UNREADABLE = "The file couldn't be read — choose it again.";

const REQUIRED_COLUMNS = ['type', 'document_number', 'posting_date', 'amount', 'currency'];
const OPTIONAL_COLUMNS = [
  'line',
  'po_number',
  'po_line',
  'accrual_month',
  'vendor_code',
  'vendor_name',
  'description',
  'swbs_code',
  'tracking_key',
  'line_item_id',
];

type Step =
  | { kind: 'choose' }
  | { kind: 'result'; request: FinancialImportRequest; check: FinancialImportCheck };

export interface ImportDialogProps {
  onClose: () => void;
}

/**
 * Import GSAP extract (015 US4, FR-004 – FR-009): choose a .csv file, MAPP checks it without
 * saving and says what the import will do — the counts, every row problem and every row that will
 * not be linked — then Import applies the valid rows. The toast gives the result; the figures on
 * every page are asked again.
 */
export function ImportDialog({ onClose }: ImportDialogProps) {
  const { show } = useToast();
  const check = useCheckImport();
  const importExtract = useImportExtract();
  const inputId = useId();
  const input = useRef<HTMLInputElement>(null);
  const form = useForm({
    resolver: zodResolver(importFileSchema),
    defaultValues: { file: null },
    ...CHECK_ON_SAVE,
  });
  const file = useWatch({ control: form.control, name: 'file' });
  const fileError = form.formState.errors.file?.message;
  const [failure, setFailure] = useState<string | null>(null);
  const [step, setStep] = useState<Step>({ kind: 'choose' });
  const busy = check.isPending || importExtract.isPending;

  function checkFile() {
    setFailure(null);
    void form.handleSubmit(({ file: chosen }) => runCheck(chosen))();
  }

  async function runCheck(chosen: File) {
    let request: FinancialImportRequest;
    try {
      request = { fileName: chosen.name, content: await readAsBase64(chosen) };
    } catch {
      form.setError('file', { message: UNREADABLE });
      return;
    }
    try {
      const result = await check.mutateAsync(request);
      setStep({ kind: 'result', request, check: result });
    } catch (error) {
      showFailure(error, true);
    }
  }

  async function runImport(request: FinancialImportRequest) {
    try {
      const result = await importExtract.mutateAsync(request);
      const { counts, fileName } = result.import;
      const parts = [importedToast(fileName, counts)];
      if (counts.relinked > 0) {
        parts.push(relinkedText(counts.relinked));
      }
      if (counts.refused > 0) {
        parts.push(refusedText(counts.refused));
      }
      show({ message: parts.join('. ') });
      onClose();
    } catch (error) {
      showFailure(error, false);
    }
  }

  /** Whole-file refusals show at the file; anything else above the form. */
  function showFailure(error: unknown, checking: boolean) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'signInRequired' || apiError.kind === 'sessionExpired') {
      onClose();
      return;
    }
    const fileProblem = apiError.fields.find(
      (field) => field.field === 'content' || field.field === 'fileName',
    );
    if (fileProblem) {
      setStep({ kind: 'choose' });
      form.setError('file', { message: fileProblem.message });
      return;
    }
    setFailure(checking ? apiError.message : `Nothing was imported. ${apiError.message}`);
  }

  function chooseAnother() {
    setStep({ kind: 'choose' });
    form.reset();
    setFailure(null);
    if (input.current) {
      input.current.value = '';
    }
  }

  async function downloadSample() {
    try {
      await downloadSampleFile();
    } catch (error) {
      show({ message: error instanceof ApiError ? error.message : UNREADABLE });
    }
  }

  const importable =
    step.kind === 'result' && step.check.counts.rows - step.check.counts.refused > 0;
  const submitLabel =
    step.kind === 'choose'
      ? 'Check file'
      : importable
        ? importButtonLabel(step.check.counts)
        : 'Choose another file';

  function submit() {
    if (busy) {
      return;
    }
    if (step.kind === 'choose') {
      checkFile();
    } else if (importable) {
      void runImport(step.request);
    } else {
      chooseAnother();
    }
  }

  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && !importExtract.isPending && onClose()}
      title="Import GSAP extract"
      description="SP&A One checks the file first and tells you what it will change. Nothing is saved until you import."
      submitLabel={submitLabel}
      busy={busy}
      onSubmit={submit}
    >
      {failure && (
        <p className={styles.failure} role="alert">
          <CircleAlert aria-hidden="true" />
          {failure}
        </p>
      )}
      {step.kind === 'choose' ? (
        <div className={styles.step}>
          <div className={styles.format}>
            <p>
              A UTF-8 .csv file with a header row, at most 700 KB and 5,000 rows. Columns can come
              in any order.
            </p>
            <p>
              <span className={styles.label}>Required columns: </span>
              <span className={styles.columns}>{REQUIRED_COLUMNS.join(', ')}</span>
            </p>
            <p>
              <span className={styles.label}>Optional columns: </span>
              <span className={styles.columns}>{OPTIONAL_COLUMNS.join(', ')}</span>
            </p>
            <div>
              <Button
                variant="tertiary"
                size="sm"
                icon={<Download />}
                onClick={() => void downloadSample()}
              >
                Download sample file
              </Button>
            </div>
          </div>
          <div className={styles.field}>
            <label htmlFor={inputId} className={styles.fieldLabel}>
              GSAP extract file <span className={styles.marking}>Required</span>
            </label>
            <input
              ref={input}
              id={inputId}
              name="file"
              type="file"
              accept=".csv,text/csv"
              className={styles.file}
              aria-invalid={fileError ? true : undefined}
              aria-describedby={fileError ? `${inputId}-error` : undefined}
              onChange={(event) => {
                const chosen = event.target.files?.[0] ?? null;
                form.setValue('file', chosen);
                if (chosen) {
                  checkFile();
                }
              }}
            />
            {fileError && (
              <p id={`${inputId}-error`} className={styles.error}>
                <CircleAlert aria-hidden="true" />
                {fileError}
              </p>
            )}
          </div>
          {check.isPending && file && (
            <p className={styles.checking} role="status">
              <LoaderCircle aria-hidden="true" className={styles.spinner} />
              {`Checking ${file.name}…`}
            </p>
          )}
        </div>
      ) : (
        <ImportCheckResult
          check={step.check}
          importable={importable}
          importing={importExtract.isPending}
          onChooseAnother={chooseAnother}
        />
      )}
    </FormDialog>
  );
}

/** The file's bytes as base64, as the API takes them. */
function readAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const url = String(reader.result);
      resolve(url.slice(url.indexOf(',') + 1));
    };
    reader.onerror = () => reject(reader.error ?? new Error('unreadable'));
    reader.readAsDataURL(file);
  });
}
