import { useId } from 'react';
import type { ComponentProps, MouseEvent } from 'react';
import { Link } from 'react-router';
import { LevelTag, StatusChip, Tooltip } from '@/components';
import { cx } from '@/lib/cx';
import { useFormatters } from '@/lib/useFormatters';
import gridStyles from './AllWorkGrid.module.css';
import type { AllWorkResult, AllWorkRow, AllWorkStatus } from './allWorkTypes';
import { barOf, hasNoDates, monthDays, todayOffset } from './calendar';
import type { Bar } from './calendar';
import styles from './CalendarView.module.css';
import { isControl } from './cellText';
import { RowName } from './RowName';
import { rowHref } from './rowHref';
import type { VisibleRow } from './viewState';

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/** The level of a bar, as a word in its tip. */
const LEVEL_WORDS: Partial<Record<AllWorkRow['level'], string>> = {
  campaign: 'Campaign',
  initiative: 'Initiative',
  activity: 'Activity',
};

export interface CalendarViewProps {
  /** The answer for view=calendar (no line-item or unassigned rows). */
  result: AllWorkResult;
  /** The rows of the page on screen (D-115). */
  rows: VisibleRow[];
  continued: string;
  search: string;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
}

/**
 * The Calendar's timeline (US5, FR-025 – FR-027): the year with its months across the top, a bar
 * per campaign, initiative and activity from its start to its end date in its status colour with
 * the status word, the Today line, cut and open ends said in words, and "No dates yet (N)".
 * Bars open their record and cannot be dragged: dates change on the record's page (FR-027).
 */
export function CalendarView({
  result,
  rows,
  continued,
  search,
  onToggle,
  onOpen,
}: CalendarViewProps) {
  const headingId = useId();
  const { year, today } = result;
  const columns = monthDays(year)
    .map((days) => `minmax(0, ${days}fr)`)
    .join(' ');
  const todayAt = todayOffset(today, year);
  const undated = result.rows.filter(hasNoDates);
  const statuses = legendOf(result.rows);

  return (
    <div className={gridStyles.grid}>
      {statuses.length > 0 && (
        <div className={styles.legend}>
          <span className={styles.legendTitle}>Status</span>
          <ul className={styles.legendList}>
            {statuses.map((status) => (
              <li key={`${status.tone}-${status.label}`}>
                <StatusChip tone={status.tone}>{status.label}</StatusChip>
              </li>
            ))}
          </ul>
        </div>
      )}
      {continued && <p className={gridStyles.continued}>Continued: {continued}</p>}
      <div className={gridStyles.scroll}>
        <table className={cx(gridStyles.table, styles.calendar)}>
          <caption className="visually-hidden">Calendar {year}</caption>
          <thead>
            <tr>
              <th scope="col" className={gridStyles.nameHead}>
                Name
              </th>
              <th scope="col" className={styles.trackHead}>
                <div className={styles.track}>
                  <div className={styles.months} style={{ gridTemplateColumns: columns }}>
                    {MONTHS.map((month) => (
                      <span key={month}>{month}</span>
                    ))}
                  </div>
                  {todayAt !== null && (
                    <span className={styles.todayLabel} style={{ left: `${todayAt}%` }}>
                      Today
                    </span>
                  )}
                </div>
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((visible) => (
              <CalendarRow
                key={visible.row.key}
                visible={visible}
                year={year}
                columns={columns}
                todayAt={todayAt}
                search={search}
                onToggle={onToggle}
                onOpen={onOpen}
              />
            ))}
          </tbody>
        </table>
      </div>
      {undated.length > 0 && (
        <section className={styles.undated} aria-labelledby={headingId}>
          <h3 id={headingId} className={styles.undatedTitle}>
            No dates yet ({undated.length})
          </h3>
          <ul className={styles.undatedList}>
            {undated.map((row) => (
              <li key={row.key} className={styles.undatedItem}>
                {row.level !== 'unassigned' && <LevelTag level={row.level} />}
                <RecordLink row={row} onOpen={onOpen} className={styles.undatedLink}>
                  {row.name}
                </RecordLink>
                {row.status && <StatusChip tone={row.status.tone}>{row.status.label}</StatusChip>}
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}

interface CalendarRowProps {
  visible: VisibleRow;
  year: number;
  /** The month columns, as wide as their days. */
  columns: string;
  todayAt: number | null;
  search: string;
  onToggle: (key: string) => void;
  onOpen: (row: AllWorkRow) => void;
}

function CalendarRow({
  visible,
  year,
  columns,
  todayAt,
  search,
  onToggle,
  onOpen,
}: CalendarRowProps) {
  const { row } = visible;
  const bar = barOf(row, year);

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
        <RowName visible={visible} search={search} showStatus onToggle={onToggle} onOpen={onOpen} />
      </th>
      <td className={styles.trackCell}>
        <div className={styles.track}>
          <div className={styles.lines} style={{ gridTemplateColumns: columns }} aria-hidden="true">
            {MONTHS.map((month) => (
              <span key={month} />
            ))}
          </div>
          {todayAt !== null && (
            <span className={styles.today} style={{ left: `${todayAt}%` }} aria-hidden="true" />
          )}
          {bar && <BarLink row={row} bar={bar} year={year} onOpen={onOpen} />}
        </div>
      </td>
    </tr>
  );
}

interface BarLinkProps {
  row: AllWorkRow;
  bar: Bar;
  year: number;
  onOpen: (row: AllWorkRow) => void;
}

/** A bar: a link to the record with its name, status and dates in a tip (FR-026, FR-027). */
function BarLink({ row, bar, year, onOpen }: BarLinkProps) {
  const { date, dateRange, money } = useFormatters();
  const tone = row.status?.tone ?? 'neutral';
  const tip = (
    <span className={styles.tip}>
      <span className={styles.tipTitle}>
        {row.name} · {row.recordId}
      </span>
      <span>
        {LEVEL_WORDS[row.level]}
        {row.status && ` · ${row.status.label}`}
      </span>
      <span>
        {bar.ongoing && row.startDate
          ? `From ${date(row.startDate)}, ongoing`
          : dateRange(row.startDate, row.endDate)}
      </span>
      {row.owner && (
        <span>
          Owner: {row.owner.name}
          {row.owner.inactive && ' (inactive)'}
        </span>
      )}
      {row.plannedCost && <span>Planned cost: {money(row.plannedCost)}</span>}
      <span>Change dates on the record&apos;s page</span>
    </span>
  );

  return (
    <Tooltip content={tip}>
      <RecordLink
        row={row}
        onOpen={onOpen}
        className={cx(
          styles.bar,
          styles[tone],
          bar.continuesFrom && styles.cutStart,
          (bar.continuesInto || bar.ongoing) && styles.openEnd,
        )}
        style={{ left: `${bar.left}%`, width: `${bar.width}%` }}
      >
        {bar.continuesFrom && <span className={styles.edge}>Continues from {year - 1}</span>}
        <span className={styles.barName}>{row.name}</span>
        {row.status && <span className={styles.barStatus}>{row.status.label}</span>}
        {bar.ongoing && <span className={styles.edge}>Ongoing</span>}
        {bar.continuesInto && <span className={styles.edge}>Continues into {year + 1}</span>}
      </RecordLink>
    </Tooltip>
  );
}

type RecordLinkProps = Omit<ComponentProps<typeof Link>, 'to' | 'onClick' | 'draggable'> & {
  row: AllWorkRow;
  onOpen: (row: AllWorkRow) => void;
};

/**
 * A link to a row's record that first checks it still exists (onOpen); never draggable, so a bar
 * cannot be dragged to other dates (FR-027).
 */
function RecordLink({ row, onOpen, ...linkProps }: RecordLinkProps) {
  function open(event: MouseEvent<HTMLAnchorElement>) {
    if (event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
      return;
    }
    event.preventDefault();
    onOpen(row);
  }
  return (
    // The other props include those of a Tooltip around the link (its ref and handlers).
    <Link {...linkProps} to={rowHref(row)} draggable={false} onClick={open} />
  );
}

/** The statuses the bars show, once each, for the legend (FR-025). */
function legendOf(rows: AllWorkRow[]): AllWorkStatus[] {
  const seen = new Map<string, AllWorkStatus>();
  for (const row of rows) {
    if (row.status && LEVEL_WORDS[row.level]) {
      const key = `${row.status.tone}:${row.status.label}`;
      if (!seen.has(key)) {
        seen.set(key, row.status);
      }
    }
  }
  return [...seen.values()];
}
