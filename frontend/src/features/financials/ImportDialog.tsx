import { CircleAlert, Download, FileText, LoaderCircle, RotateCcw } from 'lucide-react';
import { useId, useRef, useState } from 'react';
import { ApiError } from '@/api/errors';
import { Button, FormDialog, useToast } from '@/components';
import { downloadSampleFile, useCheckImport, useImportExtract } from './financialsApi';
import type {
  FinancialImportCheck,
  FinancialImportNotLinkedRow,
  FinancialImportProblem,
  FinancialImportRequest,
} from './financialsApi';
import {
  importButtonLabel,
  importCountsText,
  importedToast,
  listHeading,
  notLinkedLine,
  NOTHING_TO_IMPORT,
  problemLine,
  refusedText,
  relinkedText,
} from './financialsText';
import styles from './ImportDialog.module.css';

/** The largest file the API takes (FR-004): 700 KB. */
export const MAX_FILE_BYTES = 700 * 1024;
const TOO_LARGE = 'The file is larger than 700 KB — split it into smaller files.';
const NO_FILE = 'Choose a .csv file — GSAP extracts are imported as CSV.';
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
 * Import GSAP extract (015 US4, FR-004 – FR-009): choose a .csv file, NorthStar checks it without
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
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [failure, setFailure] = useState<string | null>(null);
  const [step, setStep] = useState<Step>({ kind: 'choose' });
  const busy = check.isPending || importExtract.isPending;

  async function runCheck(chosen: File | null) {
    setFailure(null);
    if (!chosen) {
      setFileError(NO_FILE);
      return;
    }
    if (chosen.size > MAX_FILE_BYTES) {
      setFileError(TOO_LARGE);
      return;
    }
    setFileError(null);
    let request: FinancialImportRequest;
    try {
      request = { fileName: chosen.name, content: await readAsBase64(chosen) };
    } catch {
      setFileError(UNREADABLE);
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
      setFileError(fileProblem.message);
      return;
    }
    setFailure(checking ? apiError.message : `Nothing was imported. ${apiError.message}`);
  }

  function chooseAnother() {
    setStep({ kind: 'choose' });
    setFile(null);
    setFailure(null);
    setFileError(null);
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
      void runCheck(file);
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
      description="NorthStar checks the file first and tells you what it will change. Nothing is saved until you import."
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
                setFile(chosen);
                if (chosen) {
                  void runCheck(chosen);
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
        <CheckResult
          check={step.check}
          importable={importable}
          importing={importExtract.isPending}
          onChooseAnother={chooseAnother}
        />
      )}
    </FormDialog>
  );
}

interface CheckResultProps {
  check: FinancialImportCheck;
  importable: boolean;
  importing: boolean;
  onChooseAnother: () => void;
}

/** What the import will do: the counts, then every problem and every not-linked row. */
function CheckResult({ check, importable, importing, onChooseAnother }: CheckResultProps) {
  return (
    <div className={styles.step}>
      <div className={styles.summary} role="status">
        <FileText aria-hidden="true" />
        <div>
          <p className={styles.fileName}>{check.fileName}</p>
          <p>{importCountsText(check.counts)}</p>
          {check.unusedColumns.length > 0 && (
            <p className={styles.muted}>{`Columns not used: ${check.unusedColumns.join(', ')}`}</p>
          )}
        </div>
      </div>
      {!importable && <p className={styles.nothing}>{NOTHING_TO_IMPORT}</p>}
      {check.problemsTotal > 0 && (
        <RowList<FinancialImportProblem>
          heading={listHeading('Problems', check.problems.length, check.problemsTotal)}
          rows={check.problems}
          line={problemLine}
          tone="problem"
        />
      )}
      {check.notLinkedTotal > 0 && (
        <RowList<FinancialImportNotLinkedRow>
          heading={listHeading('Not linked', check.notLinked.length, check.notLinkedTotal)}
          rows={check.notLinked}
          line={notLinkedLine}
        />
      )}
      {importing && (
        <p className={styles.checking} role="status">
          <LoaderCircle aria-hidden="true" className={styles.spinner} />
          {`Importing ${check.fileName}…`}
        </p>
      )}
      {importable && (
        <div>
          <Button variant="tertiary" size="sm" icon={<RotateCcw />} onClick={onChooseAnother}>
            Choose another file
          </Button>
        </div>
      )}
    </div>
  );
}

interface RowListProps<T> {
  heading: string;
  rows: T[];
  line: (row: T) => string;
  tone?: 'problem';
}

function RowList<T>({ heading, rows, line, tone }: RowListProps<T>) {
  const id = useId();
  return (
    <section aria-labelledby={id} className={styles.rows}>
      <h3 id={id} className={tone === 'problem' ? styles.problemHeading : styles.heading}>
        {heading}
      </h3>
      {/* A long list scrolls; it can take the focus, so the keyboard can scroll it too. */}
      <ul className={styles.list} tabIndex={0} aria-labelledby={id}>
        {rows.map((row) => {
          const text = line(row);
          return <li key={text}>{text}</li>;
        })}
      </ul>
    </section>
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
