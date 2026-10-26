import type { MouseEvent } from 'react';
import { cx } from '@/ui/cx';
import gridStyles from './AllWorkGrid.module.css';
import type { AllWorkRow } from './all-work-types';
import { BarLink } from './BarLink';
import { barOf } from './calendar';
import styles from './CalendarView.module.css';
import { isControl } from './cell-text';
import { RowName } from './RowName';
import type { VisibleRow } from './view-rows';

export interface CalendarRowProps {
  visible: VisibleRow;
  year: number;
  /** The last year of the timeline (the same as year for one year). */
  yearTo: number;
  /** The month or quarter columns, as wide as their days. */
  columns: string;
  /** How many month or quarter columns there are. */
  count: number;
  todayAt: number | null;
  /** The narrowest the whole timeline gets, in pixels. */
  trackPx: number;
  search: string;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
}

/** One row of the Calendar: its name and its bar on the timeline. */
export function CalendarRow({
  visible,
  year,
  yearTo,
  columns,
  count,
  todayAt,
  trackPx,
  search,
  onToggle,
  onOpen,
}: CalendarRowProps) {
  const { row } = visible;
  const bar = barOf(row, year, yearTo);

  function openFromRow(event: MouseEvent<HTMLTableRowElement>) {
    if (!isControl(event.target)) {
      onOpen(row);
    }
  }

  return (
    <tr
      data-row-key={row.key}
      className={cx(gridStyles.row, row.role === 'context' && gridStyles.context)}
      onClick={openFromRow}
    >
      <th scope="row" className={cx(gridStyles.nameCell, gridStyles[row.level])}>
        {/* The bar says the status; only a row without a bar shows its chip (UI audit AW-06). */}
        <RowName
          visible={visible}
          search={search}
          showStatus={!bar}
          onToggle={onToggle}
          onOpen={onOpen}
        />
      </th>
      <td className={styles.trackCell}>
        <div className={styles.track}>
          <div className={styles.lines} style={{ gridTemplateColumns: columns }} aria-hidden="true">
            {Array.from({ length: count }, (_, index) => (
              <span key={index} />
            ))}
          </div>
          {todayAt !== null && (
            <span className={styles.today} style={{ left: `${todayAt}%` }} aria-hidden="true" />
          )}
          {bar && (
            <BarLink
              row={row}
              bar={bar}
              year={year}
              yearTo={yearTo}
              trackPx={trackPx}
              onOpen={onOpen}
            />
          )}
        </div>
      </td>
    </tr>
  );
}
