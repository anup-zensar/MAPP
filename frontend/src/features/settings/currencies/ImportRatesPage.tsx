import { CircleAlert, Download } from 'lucide-react';
import { useId, useRef, useState } from 'react';
import type { ChangeEvent } from 'react';
import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { useToast } from '@/ui/Toast/useToast';
import { usePageTitle } from '@/ui/usePageTitle';
import { CURRENCIES_PATH } from './currency-paths';
import { checkRateImport, downloadRateSample, useImportRates } from './rates-api';
import type { RateImportCheck, RateImportUpload } from './rates-api';
import { FILE_TOO_LARGE, importedMessage, MAX_FILE_BYTES, NOT_A_CSV } from './rate-words';
import styles from './rates.module.css';
import { ReviewCard } from './ReviewCard';

/**
 * Import GSAP rates (016 US1, FR-006 – FR-008), at /settings/currencies/import: choose the GSAP
 * daily rate extract (a .csv of up to 2 MB, simulated — MAPP reads the file and connects to
 * no system); before anything is saved, the review lists every row's outcome with its row number,
 * grouped and counted; **Import n rates** saves the rows to add and replace in one step and
 * **Cancel** saves nothing. **Download a sample file** gives a file of today's rates.
 */
export function ImportRatesPage() {
  usePageTitle('Import GSAP rates');
  const navigate = useNavigate();
  const { show } = useToast();
  const importRates = useImportRates();
  const fileInput = useRef<HTMLInputElement>(null);
  const inputId = useId();
  const [upload, setUpload] = useState<RateImportUpload | null>(null);
  const [review, setReview] = useState<RateImportCheck | null>(null);
  const [reviewing, setReviewing] = useState(false);
  const [fileError, setFileError] = useState<string | null>(null);
  const [importFailure, setImportFailure] = useState<string | null>(null);

  function reset() {
    setUpload(null);
    setReview(null);
    setFileError(null);
    setImportFailure(null);
    if (fileInput.current) {
      fileInput.current.value = '';
    }
  }

  async function choose(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    setReview(null);
    setUpload(null);
    setImportFailure(null);
    if (!file) {
      setFileError(null);
      return;
    }
    if (!file.name.toLowerCase().endsWith('.csv')) {
      setFileError(NOT_A_CSV);
      return;
    }
    if (file.size > MAX_FILE_BYTES) {
      setFileError(FILE_TOO_LARGE);
      return;
    }
    setFileError(null);
    setReviewing(true);
    try {
      const chosen = { fileName: file.name, content: await base64Of(file) };
      setReview(await checkRateImport(chosen));
      setUpload(chosen);
    } catch (error) {
      const message = failureText(error);
      if (message) {
        setFileError(message);
      }
    } finally {
      setReviewing(false);
    }
  }

  async function runImport() {
    if (!upload || importRates.isPending) {
      return;
    }
    setImportFailure(null);
    try {
      const done = await importRates.mutateAsync(upload);
      show({ message: importedMessage(done.added + done.replaced, done.fileName) });
      void navigate(CURRENCIES_PATH);
    } catch (error) {
      const message = failureText(error);
      if (message) {
        setImportFailure(message);
      }
    }
  }

  async function downloadSample() {
    try {
      await downloadRateSample();
    } catch (error) {
      const message = failureText(error);
      if (message) {
        show({ message });
      }
    }
  }

  const fileErrorId = `${inputId}-error`;
  const hintId = `${inputId}-hint`;

  return (
    <>
      <PageHeader
        breadcrumb={[
          { label: 'Settings', to: '/settings' },
          { label: 'Currencies', to: CURRENCIES_PATH },
          { label: 'Import GSAP rates' },
        ]}
        title="Import GSAP rates"
        description="Import the GSAP daily rate extract until GSAP sends rates by itself. MAPP reads the file and connects to no system."
      />
      <div className={styles.stack}>
        <Card title="Rate file">
          <div className={styles.upload}>
            <label htmlFor={inputId} className={styles.label}>
              GSAP daily rate extract (.csv)
            </label>
            <p id={hintId} className={styles.note}>
              Up to 2 MB, with the columns rate_type, from_currency, to_currency, valid_from and
              rate. Only M rates quoted from USD are used; currencies are never created.
            </p>
            <div className={styles.uploadRow}>
              <input
                ref={fileInput}
                id={inputId}
                type="file"
                accept=".csv,text/csv"
                className={styles.fileInput}
                aria-describedby={[hintId, fileError && fileErrorId].filter(Boolean).join(' ')}
                aria-invalid={fileError ? true : undefined}
                onChange={(event) => void choose(event)}
              />
              <Button
                variant="tertiary"
                size="sm"
                icon={<Download />}
                onClick={() => void downloadSample()}
              >
                Download a sample file
              </Button>
            </div>
            {reviewing && (
              <p role="status" className={styles.note}>
                Reviewing the file…
              </p>
            )}
            {fileError && (
              <p id={fileErrorId} role="alert" className={styles.error}>
                <CircleAlert aria-hidden="true" />
                {fileError}
              </p>
            )}
          </div>
        </Card>
        {review && (
          <ReviewCard
            review={review}
            importing={importRates.isPending}
            failure={importFailure}
            onImport={() => void runImport()}
            onCancel={reset}
          />
        )}
      </div>
    </>
  );
}

/** The API's plain-language message; nothing when the session has ended (the sign-in page says so). */
function failureText(error: unknown): string | null {
  const failure = error instanceof ApiError ? error : new ApiError('other');
  if (failure.kind === 'signInRequired' || failure.kind === 'sessionExpired') {
    return null;
  }
  const [first] = failure.fields;
  return failure.kind === 'validation' && first ? first.message : failure.message;
}

/** The file's bytes in base64, as the API takes them. */
function base64Of(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const url = String(reader.result);
      resolve(url.slice(url.indexOf(',') + 1));
    };
    reader.onerror = () => reject(reader.error ?? new Error('The file could not be read.'));
    reader.readAsDataURL(file);
  });
}
