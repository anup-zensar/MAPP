import { FileText, LoaderCircle, RotateCcw } from 'lucide-react';
import { useId } from 'react';
import { Button } from '@/ui/Button/Button';
import type {
  FinancialImportCheck,
  FinancialImportNotLinkedRow,
  FinancialImportProblem,
} from './api';
import {
  importCountsText,
  listHeading,
  notLinkedLine,
  NOTHING_TO_IMPORT,
  problemLine,
} from './financials-text';
import styles from './ImportDialog.module.css';

interface ImportCheckResultProps {
  check: FinancialImportCheck;
  importable: boolean;
  importing: boolean;
  onChooseAnother: () => void;
}

/** What the import will do: the counts, then every problem and every not-linked row. */
export function ImportCheckResult({
  check,
  importable,
  importing,
  onChooseAnother,
}: ImportCheckResultProps) {
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
      {/* eslint-disable-next-line jsx-a11y-x/no-noninteractive-tabindex -- the list scrolls */}
      <ul className={styles.list} tabIndex={0} aria-labelledby={id}>
        {rows.map((row) => {
          const text = line(row);
          return <li key={text}>{text}</li>;
        })}
      </ul>
    </section>
  );
}
