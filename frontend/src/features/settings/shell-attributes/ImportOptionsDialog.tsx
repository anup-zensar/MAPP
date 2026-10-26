import { LoaderCircle } from 'lucide-react';
import { Dialog } from 'radix-ui';
import { useId, useRef, useState } from 'react';
import type { ChangeEvent } from 'react';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { useToast } from '@/ui/Toast/useToast';
import { formatWhole } from '@/format/decimal';
import { midSentence, plural } from '@/format/text';
import { checkOptionImport, useImportOptions } from './shellAttributesApi';
import type {
  ImportRow,
  OptionImportCheck,
  OptionImportRequest,
  ShellAttribute,
} from './shellAttributesApi';
import styles from './ImportOptionsDialog.module.css';

/** The largest file the API takes: 700 KB, which is about 950 KB once encoded (plan, D-051). */
const MAX_FILE_BYTES = 700 * 1024;
/** Each list of the review shows this many rows, then "and <n> more". */
const ROWS_SHOWN = 100;

const WRONG_TYPE =
  'Choose a .csv or .xlsx file. An older .xls file can be saved as .xlsx in Excel.';
const TOO_LARGE = 'The file is larger than 700 KB — split it into smaller files.';
const UNREADABLE = "This file can't be read. Save it again as .csv or .xlsx and try again.";
const NOTHING_TO_ADD = 'Nothing to add — every row is already an option or has a problem.';

export interface ImportOptionsDialogProps {
  attribute: ShellAttribute;
  onClose: () => void;
}

/**
 * Import options into <label> (feature 002, US6, FR-031): the person chooses a .csv or .xlsx file
 * made from the template; the API checks every row and the dialog lists the outcome — the options
 * to add, the rows already present (skipped) and the rows with problems, each with its row number
 * — before anything is added. **Add <n> options** adds only the new valid ones; Cancel adds
 * nothing. Nothing is ever removed, renamed or reactivated.
 */
export function ImportOptionsDialog({ attribute, onClose }: ImportOptionsDialogProps) {
  const { show } = useToast();
  const importOptions = useImportOptions();
  const fileFieldId = useId();
  const fileProblemId = useId();
  const focusBeforeOpen = useRef<HTMLElement | null>(null);
  // The file as the API takes it, kept for Add once it is checked.
  const [file, setFile] = useState<OptionImportRequest | null>(null);
  const [check, setCheck] = useState<OptionImportCheck | null>(null);
  const [checking, setChecking] = useState(false);
  // What is wrong with the chosen file (type, size, or what the API found).
  const [fileProblem, setFileProblem] = useState<string | null>(null);
  // A failure that is not about the file: the server could not be reached, for example.
  const [failure, setFailure] = useState<string | null>(null);
  const parent = attribute.parentAttribute;

  async function chooseFile(event: ChangeEvent<HTMLInputElement>) {
    const chosen = event.target.files?.[0];
    setCheck(null);
    setFile(null);
    setFileProblem(null);
    setFailure(null);
    if (!chosen) {
      return;
    }
    const name = chosen.name.toLowerCase();
    if (!name.endsWith('.csv') && !name.endsWith('.xlsx')) {
      setFileProblem(WRONG_TYPE);
      return;
    }
    if (chosen.size > MAX_FILE_BYTES) {
      setFileProblem(TOO_LARGE);
      return;
    }
    setChecking(true);
    try {
      const request = { fileName: chosen.name, content: await readAsBase64(chosen) };
      setCheck(await checkOptionImport(attribute.id, request));
      setFile(request);
    } catch (error) {
      showProblem(error);
    } finally {
      setChecking(false);
    }
  }

  async function add() {
    if (!file || importOptions.isPending) {
      return;
    }
    setFailure(null);
    try {
      const result = await importOptions.mutateAsync({ attributeId: attribute.id, file });
      show({ message: `${plural(result.added, 'option')} added to ${attribute.label}` });
      onClose();
    } catch (error) {
      showProblem(error);
    }
  }

  /** The file's problems go under the file field; anything else above the buttons. */
  function showProblem(error: unknown) {
    if (!(error instanceof ApiError)) {
      setFileProblem(UNREADABLE);
      return;
    }
    if (error.kind === 'signInRequired' || error.kind === 'sessionExpired') {
      return;
    }
    if (error.kind === 'validation' && error.fields.length > 0) {
      setCheck(null);
      setFile(null);
      setFileProblem(error.fields.map((field) => field.message).join(' '));
      return;
    }
    setFailure(error.message);
  }

  return (
    <Dialog.Root open onOpenChange={(open) => !open && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content
          className={styles.dialog}
          onOpenAutoFocus={() => {
            focusBeforeOpen.current =
              document.activeElement instanceof HTMLElement ? document.activeElement : null;
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault();
            focusBeforeOpen.current?.focus();
          }}
        >
          <Dialog.Title className={styles.title}>
            {`Import options into ${attribute.label}`}
          </Dialog.Title>
          <Dialog.Description className={styles.text}>
            {`Download the template, fill one row per option${
              parent ? `, with its ${midSentence(parent.name)} option in the second column` : ''
            }, then upload it. Nothing is added until you confirm.`}
          </Dialog.Description>

          <div className={styles.body}>
            <div className={styles.fileField}>
              <label htmlFor={fileFieldId} className={styles.label}>
                Choose a file
              </label>
              <input
                id={fileFieldId}
                type="file"
                accept=".csv,.xlsx"
                className={styles.file}
                aria-invalid={fileProblem ? true : undefined}
                aria-describedby={fileProblem ? fileProblemId : undefined}
                onChange={(event) => void chooseFile(event)}
              />
              {fileProblem && (
                <p id={fileProblemId} className={styles.problem}>
                  {fileProblem}
                </p>
              )}
            </div>

            {checking && (
              <p role="status" className={styles.checking}>
                <LoaderCircle aria-hidden="true" className={styles.spinner} />
                <span className="visually-hidden">Loading…</span>
              </p>
            )}
            {check && <Review check={check} dependent={parent !== null} />}
            {failure && (
              <p role="alert" className={styles.problem}>
                {failure}
              </p>
            )}
          </div>

          <div className={styles.actions}>
            <Dialog.Close asChild>
              <Button>Cancel</Button>
            </Dialog.Close>
            {check && check.toAdd.length > 0 && (
              <Button
                variant="primary"
                icon={
                  importOptions.isPending ? <LoaderCircle className={styles.spinner} /> : undefined
                }
                aria-busy={importOptions.isPending || undefined}
                onClick={() => void add()}
              >
                {`Add ${plural(check.toAdd.length, 'option')}`}
              </Button>
            )}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

interface ReviewProps {
  check: OptionImportCheck;
  /** The attribute depends on another: rows show their parent option too. */
  dependent: boolean;
}

/** Every row's outcome at once, in three lists with their row numbers (US6 scenario 1). */
function Review({ check, dependent }: ReviewProps) {
  const toAdd = `${plural(check.toAdd.length, 'option')} to add`;
  const present = `${formatWhole(check.alreadyPresent.length)} already present (skipped)`;
  const problems = `${plural(check.problems.length, 'row')} with problems`;

  return (
    <div className={styles.review}>
      {/* Screen readers hear the outcome once; the lists are there to read at leisure. */}
      <p role="status" className="visually-hidden">
        {`${toAdd} · ${present} · ${problems}`}
      </p>
      {check.toAdd.length === 0 && <p className={styles.nothing}>{NOTHING_TO_ADD}</p>}
      <RowList summary={toAdd} rows={check.toAdd} dependent={dependent} />
      <RowList summary={present} rows={check.alreadyPresent} dependent={dependent} />
      <RowList summary={problems} rows={check.problems} dependent={dependent} open problem />
    </div>
  );
}

interface RowListProps {
  summary: string;
  rows: ImportRow[];
  dependent: boolean;
  /** Shown unfolded at first. */
  open?: boolean;
  /** Its messages are problems. */
  problem?: boolean;
}

/** One outcome: its count, then — unfolded — its first 100 rows, "Row 12 · CN04000220 · …". */
function RowList({ summary, rows, dependent, open = false, problem = false }: RowListProps) {
  if (rows.length === 0) {
    return <p className={styles.empty}>{summary}</p>;
  }
  const more = rows.length - ROWS_SHOWN;
  return (
    <details className={styles.list} open={open}>
      <summary className={styles.summary}>{summary}</summary>
      <ul role="list" className={styles.rows}>
        {rows.slice(0, ROWS_SHOWN).map((row) => (
          <li key={`${row.row}-${row.label}-${row.parentLabel}`} className={styles.row}>
            <span className={styles.rowNumber}>{`Row ${formatWhole(row.row)}`}</span>
            <span className={styles.rowLabel}>
              {row.label || '—'}
              {dependent && row.parentLabel && (
                <span className={styles.parent}>{` (${row.parentLabel})`}</span>
              )}
            </span>
            {row.message && (
              <span className={problem ? styles.rowProblem : styles.rowMessage}>{row.message}</span>
            )}
          </li>
        ))}
      </ul>
      {more > 0 && <p className={styles.more}>{`and ${formatWhole(more)} more`}</p>}
    </details>
  );
}

/** The file's bytes as base64, as the API takes them. */
function readAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = String(reader.result);
      resolve(dataUrl.slice(dataUrl.indexOf(',') + 1));
    };
    reader.onerror = () => reject(reader.error ?? new Error('unreadable'));
    reader.readAsDataURL(file);
  });
}
