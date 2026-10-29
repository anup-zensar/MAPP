import { Download } from 'lucide-react';
import { Button } from '@/ui/Button/Button';
import type { ImportCheck, ImportProblem } from './api';
import { downloadProblems } from './files';
import styles from './workbook.module.css';
import { LEVEL_PLURALS, changeSummary, checkSummary } from './workbook-text';

/** What an import would do per level, and every problem of every row at once (FR-031 – FR-033). */
export function CheckResult({ check }: { check: ImportCheck }) {
  return (
    <div aria-live="polite">
      <p className={styles.summary}>{checkSummary(check)}</p>
      <p className={styles.text}>{changeSummary(check)}</p>
      <div className={styles.tableScroll}>
        <table className={styles.table}>
          <caption className="visually-hidden">What the import does per level</caption>
          <thead>
            <tr>
              <th scope="col">Level</th>
              <th scope="col" className={styles.number}>
                To create
              </th>
              <th scope="col" className={styles.number}>
                To update
              </th>
              <th scope="col" className={styles.number}>
                Unchanged
              </th>
              <th scope="col" className={styles.number}>
                Left out
              </th>
            </tr>
          </thead>
          <tbody>
            {check.levels.map((l) => (
              <tr key={l.level}>
                <th scope="row">{LEVEL_PLURALS[l.level]}</th>
                <td className={styles.number}>{l.create}</td>
                <td className={styles.number}>{l.update}</td>
                <td className={styles.number}>{l.unchanged}</td>
                <td className={styles.number}>{l.leftOut}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {check.ignoredColumns.length > 0 && (
        <p className={styles.note}>
          These columns aren&apos;t used and were ignored: {check.ignoredColumns.join(', ')}
        </p>
      )}
      {check.needMoreFields > 0 && (
        <p className={styles.note}>
          {check.needMoreFields} records will need more fields before submitting or publishing.
        </p>
      )}
      {check.problems.length > 0 && (
        <ProblemsTable problems={check.problems} fileName={check.fileName} />
      )}
    </div>
  );
}

export function ProblemsTable({
  problems,
  fileName,
}: {
  problems: ImportProblem[];
  fileName: string;
}) {
  return (
    <>
      <div className={styles.row}>
        <h3 className={styles.label}>Problems ({problems.length})</h3>
        <Button
          variant="tertiary"
          size="sm"
          icon={<Download />}
          onClick={() => downloadProblems(problems, fileName)}
        >
          Download problem list
        </Button>
      </div>
      <div className={styles.tableScroll}>
        <table className={styles.table}>
          <caption className="visually-hidden">Problems</caption>
          <thead>
            <tr>
              <th scope="col">Sheet</th>
              <th scope="col" className={styles.number}>
                Row
              </th>
              <th scope="col">Column</th>
              <th scope="col">Value found</th>
              <th scope="col">Reason</th>
            </tr>
          </thead>
          <tbody>
            {problems.map((p, i) => (
              <tr key={`${p.sheet}-${p.row}-${i}`}>
                <td>{p.sheet}</td>
                <td className={styles.number}>{p.row}</td>
                <td>{p.column}</td>
                <td>{p.value}</td>
                <td>{p.reason}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
