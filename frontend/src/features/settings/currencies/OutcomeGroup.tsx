import { ChevronRight } from 'lucide-react';
import { formatRate, formatWhole } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import settingsStyles from '@/ui/settings-table.module.css';
import type { RateImportOutcome, RateImportRow } from './rates-api';
import { OUTCOME_WORDS } from './rate-words';
import styles from './rates.module.css';

/** A group of the review lists at most this many rows; the counts above cover all of them. */
const ROWS_SHOWN = 200;

const OPEN_AT_FIRST: RateImportOutcome[] = ['problem', 'notUsed', 'replace'];

export interface OutcomeGroupProps {
  outcome: RateImportOutcome;
  rows: RateImportRow[];
}

export function OutcomeGroup({ outcome, rows }: OutcomeGroupProps) {
  const { date } = useFormatters();
  const shown = rows.slice(0, ROWS_SHOWN);
  const withNote = rows.some((row) => row.message || row.previousRate);

  return (
    <details className={styles.group} open={OPEN_AT_FIRST.includes(outcome)}>
      <summary>
        {/* The open/closed marker the flex summary loses; it turns when open. */}
        <ChevronRight aria-hidden="true" className={styles.marker} />
        {`${OUTCOME_WORDS[outcome]} (${formatWhole(rows.length)})`}
      </summary>
      <div className={styles.groupBody}>
        {/* Nothing in the table takes the focus: the scrolling area does, for the keyboard. */}
        <div
          className={settingsStyles.tableScroll}
          role="region"
          aria-label={`${OUTCOME_WORDS[outcome]}: rows`}
          tabIndex={0}
        >
          <table className={settingsStyles.table}>
            <thead>
              <tr>
                <th scope="col" className={settingsStyles.number}>
                  Row
                </th>
                <th scope="col">Currency</th>
                <th scope="col">Date</th>
                <th scope="col" className={settingsStyles.number}>
                  Rate per 1 USD
                </th>
                {withNote && <th scope="col">Note</th>}
              </tr>
            </thead>
            <tbody>
              {shown.map((row) => (
                <tr key={row.row}>
                  <th scope="row" className={settingsStyles.number}>
                    {formatWhole(row.row)}
                  </th>
                  <td>{row.currency ?? '—'}</td>
                  <td className={settingsStyles.nowrap}>{row.date ? date(row.date) : '—'}</td>
                  <td className={settingsStyles.number}>{row.rate ? formatRate(row.rate) : '—'}</td>
                  {withNote && <td>{noteOf(row)}</td>}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {rows.length > shown.length && (
          <p className={styles.more}>
            {`Showing the first ${formatWhole(shown.length)} of ${formatWhole(rows.length)} rows.`}
          </p>
        )}
      </div>
    </details>
  );
}

/** "0.9214 → 0.9230" for a rate that replaces another; why, for rows not used or with a problem. */
function noteOf(row: RateImportRow): string {
  if (row.previousRate && row.rate) {
    return `${formatRate(row.previousRate)} → ${formatRate(row.rate)}`;
  }
  return row.message ?? '';
}
