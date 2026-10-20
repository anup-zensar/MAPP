import { Card, ErrorState, Skeleton } from '@/components';
import { formatWhole } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import settingsStyles from '../shared/SettingsTable.module.css';
import { RECENT_IMPORTS, useRateImports } from './ratesApi';
import type { RateImport } from './ratesApi';
import styles from './rates.module.css';

/**
 * **Recent imports** on the Currencies page (016 FR-007): the last 10 GSAP rate files with who
 * imported them and when, the dates of their rates and the row counts of their review.
 */
export function RecentImports() {
  const imports = useRateImports();
  const { date, dateTime, dateRange } = useFormatters();

  return (
    <Card title="Recent imports" subtitle={`The last ${RECENT_IMPORTS} GSAP rate files imported`}>
      {!imports.data ? (
        imports.isError ? (
          <ErrorState error={imports.error} onRetry={() => void imports.refetch()} />
        ) : (
          <Skeleton lines={3} height="18px" />
        )
      ) : imports.data.items.length === 0 ? (
        <p className={styles.cardNote}>No GSAP rate file has been imported yet.</p>
      ) : (
        // Nothing in the table takes the focus: the scrolling area does, for the keyboard.
        <div
          className={settingsStyles.tableScroll}
          role="region"
          aria-label="Recent imports"
          tabIndex={0}
        >
          <table className={settingsStyles.table}>
            <thead>
              <tr>
                <th scope="col">File</th>
                <th scope="col">Imported</th>
                <th scope="col">Rates for</th>
                <th scope="col" className={settingsStyles.number}>
                  Added
                </th>
                <th scope="col" className={settingsStyles.number}>
                  Replaced
                </th>
                <th scope="col" className={settingsStyles.number}>
                  Already present
                </th>
                <th scope="col" className={settingsStyles.number}>
                  Not used
                </th>
                <th scope="col" className={settingsStyles.number}>
                  Problems
                </th>
              </tr>
            </thead>
            <tbody>
              {imports.data.items.map((item) => (
                <ImportRow
                  key={item.id}
                  item={item}
                  when={dateTime}
                  range={(first, last) => (first === last ? date(first) : dateRange(first, last))}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

interface ImportRowProps {
  item: RateImport;
  when: (timestamp: string) => string;
  /** "21 Sep – 20 Oct 2026", or one date when the file had rates of one day. */
  range: (start: string, end: string) => string;
}

function ImportRow({ item, when, range }: ImportRowProps) {
  return (
    <tr>
      <th scope="row" className={settingsStyles.nowrap}>
        {item.fileName}
      </th>
      <td className={settingsStyles.nowrap}>
        {item.importedBy.name}
        <span className={settingsStyles.secondLine}>{when(item.importedAt)}</span>
      </td>
      <td className={settingsStyles.nowrap}>
        {item.firstDate && item.lastDate
          ? range(item.firstDate, item.lastDate)
          : 'No rates changed'}
      </td>
      <td className={settingsStyles.number}>{formatWhole(item.added)}</td>
      <td className={settingsStyles.number}>{formatWhole(item.replaced)}</td>
      <td className={settingsStyles.number}>{formatWhole(item.unchanged)}</td>
      <td className={settingsStyles.number}>{formatWhole(item.notUsed)}</td>
      <td className={settingsStyles.number}>{formatWhole(item.problems)}</td>
    </tr>
  );
}
